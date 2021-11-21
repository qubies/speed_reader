package main

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"strings"

	// "io/ioutil"
	"html/template"
	"os"

	"github.com/joho/godotenv"

	// "encoding/json"
	"github.com/gin-gonic/gin"

	"fmt"
	// "math/rand"
	"encoding/gob"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	data "github.com/qubies/speed_reader/data"
	"github.com/qubies/speed_reader/stories"
)

const (
	// the session var that holds the user's info
	userkey          = "user"
	NUMBER_OF_GROUPS = 2
)

// globals
var templates *template.Template
var system = data.Build_System("./data/experimental.db", "./data/common_words.json")

type StoryPage struct {
	User  *data.User
	State *data.Status
}

func sendInvalid(c *gin.Context) {
	c.JSON(401, gin.H{"code": "UNAUTHORIZED", "message": "There was a problem with your login, please verify that you are logged in."})
}

func validateUser(c *gin.Context) (*data.User, error) {
	session := sessions.Default(c)
	user := session.Get(userkey).(*data.User)
	if !system.Validate_User(user.User_ID, user.User_ID) {
		sendInvalid(c)
		return nil, errors.New("User is invaild")
	}
	return user, nil
}

// Routes
func introHandler(c *gin.Context) {
	c.HTML(200, "intro.html", nil)
}

func login(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := strings.TrimSpace(c.PostForm("password"))
	session := sessions.Default(c)

	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	if !system.Validate_User(username, password) {
		sendInvalid(c)
		return
	}

	session.Set(userkey, system.Users[username])
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		fmt.Println(err)
		return
	}
	c.Redirect(http.StatusFound, "/private/story")
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session.Delete(userkey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func storyStartRoute(c *gin.Context) {
	user, err := validateUser(c)
	if err != nil {
		return
	}

	userState := system.GetCurrentEvent(user)

	// verify that the user should be here....
	if userState.Event == "quiz" {
		fmt.Println("Moving user back to quiz")
		system.Record_Action(user, "User sent to quiz from story Redirect")
		c.Redirect(http.StatusFound, "/private/quiz")
		return // im not sure if the return is required here....
	}

	if userState.Event == "questionnaire" {
		fmt.Println("Moving user to questionnaire")
		system.Record_Action(user, "User sent to questionnire from story Redirect")
		c.Redirect(http.StatusFound, "/private/questionnaire")
		return // im not sure if the return is required here....
	}

	// check if they are done all the stories
	if userState.Completed {
		c.HTML(200, "experimentComplete.html", nil)
		return
	}

	SP := StoryPage{user, userState}
	c.HTML(200, "story.html", SP)
}

type storyEndPost struct {
	StartDate int
	EndDate   int
	Wpm       float32
}

func storyEndRoute(c *gin.Context) {
	user, err := validateUser(c)
	if err != nil {
		return
	}

	var record storyEndPost
	if err := c.ShouldBindJSON(&record); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
		return
	}
	system.Finish_Reading(user, record.StartDate, record.EndDate, record.Wpm)
	update_user(user, c)
}

func update_user(user *data.User, c *gin.Context) {
	session := sessions.Default(c)
	session.Set(userkey, user)
	if err := session.Save(); err != nil {
		fmt.Println(err)
	}
	nu, _ := system.User_From_ID(user.User_ID) //lazy refresh of counters
	*user = *nu
}

type actionPost struct {
	Action int
	Date   int
}

func actionRoute(c *gin.Context) {
	user, err := validateUser(c)
	if err != nil {
		return
	}
	// collect post data
	var data actionPost
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
		return
	}
	system.Record_Action(user, data.Action, data.Date)
	fmt.Println("updated", user.User_ID, data.Action)
}

type QuizStruct struct {
	Questions []stories.Question
	Name      string
	Answers   []int
	Version   int
}

// DeepCopy deepcopies a to b using json marshaling
func DeepCopy(a, b interface{}) {
	byt, _ := json.Marshal(a)
	json.Unmarshal(byt, b)
}

func quizStartRoute(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("Answers")
	user, err := validateUser(c)
	if err != nil {
		return
	}
	if user.Current_Quiz_Index == user.Current_Story_Index {
		fmt.Println("Moving user back to story")
		c.Redirect(http.StatusFound, "/private/story")
		return
	}
	s, err := system.GetQuiz(user)
	if err != nil {
		return
	}

	quiz := new(QuizStruct)
	// copy the questions from the quiz to the struct
	DeepCopy(s.Questions, &quiz.Questions)
	//shuffle the question order
	rand.Shuffle(len(quiz.Questions), func(i, j int) { quiz.Questions[i], quiz.Questions[j] = quiz.Questions[j], quiz.Questions[i] })
	quiz.Name = s.Name
	fmt.Println("Initial Answers:", quiz.Answers)
	for i := range quiz.Questions {
		qs := quiz.Questions[i].Choices
		ans_index := quiz.Questions[i].Answer
		ans := quiz.Questions[i].Choices[ans_index]
		rand.Shuffle(len(qs), func(i, j int) { qs[i], qs[j] = qs[j], qs[i] })
		for i, q := range qs {
			if q == ans {
				quiz.Answers = append(quiz.Answers, i)
				break
			}
		}
	}
	fmt.Println("Middle Answers:", quiz.Answers)
	session.Set("Answers", quiz.Answers)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		fmt.Println(err)
		return
	}
	quiz.Version = rand.Int()
	fmt.Println("Quiz", quiz)
	c.HTML(200, "quiz.html", &quiz)
}

type quizEndPost struct {
	StartDate     int
	EndDate       int
	ChosenAnswers []int
}

func quizEndRoute(c *gin.Context) {
	user, err := validateUser(c)
	if err != nil {
		return
	}

	var record quizEndPost
	if err := c.ShouldBindJSON(&record); err != nil {
		fmt.Println("Error: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
		return
	}
	session := sessions.Default(c)
	expected := session.Get("Answers").([]int)
	score := 0
	if len(expected) != len(record.ChosenAnswers) {
		fmt.Println("error, expected answers length different from recieved")
	}
	for i := range expected {
		if expected[i] == record.ChosenAnswers[i] {
			score += 1
		}
	}
	fmt.Println("quiz record:", record)
	fmt.Println("expected", expected)
	fmt.Println("Score:", score)
	system.Finish_Quiz(user, record.StartDate, record.EndDate, score)
	update_user(user, c)
	c.JSON(http.StatusOK, score)
}

// AuthRequired is a middleware to check the session
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		// Abort the request with the appropriate error code
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}

// do all of the goodness setup stuffs
func init() {
	//rand.Seed(time.Now().Unix())

	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	//load in all of the template files
	templates = template.Must(template.ParseGlob("pages/*"))

}

func main() {
	defer system.Close() // shutdown the threads

	gob.Register(new(data.User)) //teach it to serialize
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "80"
	}

	app := gin.Default()
	app.Use(sessions.Sessions("focus", sessions.NewCookieStore([]byte("secret"))))
	app.SetHTMLTemplate(templates)

	// static routing
	app.Static("/css", "./css")
	app.Static("/scripts", "./scripts")

	// so there currently aren't images, but if we want....
	// app.Static("/images","./images")

	app.GET("/", introHandler)
	app.GET("/newaccount", newAccount)
	app.POST("/login", login)
	app.GET("/logout", logout)

	private := app.Group("/private")
	private.Use(AuthRequired)
	{
		private.GET("/story", storyStartRoute)
		private.POST("/storyend", storyEndRoute)
		private.GET("/quiz", quizStartRoute)
		private.POST("/quizend", quizEndRoute)
		private.POST("/action", actionRoute)
	}
	app.Run(":" + PORT)
}
