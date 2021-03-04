package main

import (
	"errors"
	"log"
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
    userkey = "user"
    NUMBER_OF_GROUPS = 1
)

// globals 
var templates *template.Template
var system = data.Build_System("./data/focused_reader.db", "./stories/stories", "./data/common_words.json", NUMBER_OF_GROUPS)


type StoryPage struct {
    Title string
    User string
    CommonWords []string
    Story [][]string
    Spans [][]int
}

// type Choice struct {
//     Correct string   `json:"correct"`
//     Wrong   []string `json:"wrong"`
// }

// type Question struct {
//     QuestionString string `json:"question_string"`
//     Choices Choice `json:"choices"`
// }

// func new_question(question string, correct_answer string, wrong_answer []string) *Question {
//     return &Question{question, Choice{correct_answer, wrong_answer}}
// }

// type Quiz struct {
//     Name string
//     Questions []*Question
// }

func sendInvalid(c *gin.Context) {
    c.JSON(401, gin.H{"code": "UNAUTHORIZED", "message": "There was a problem with your login, please verify that you are logged in."})
}

func validateUser(c* gin.Context) (*data.User, error) {
    session := sessions.Default(c)
    user := session.Get(userkey).(*data.User)
    if !system.Validate_User(user) {
        sendInvalid(c)
        return nil, errors.New("User is invaild")
    }
    return user, nil
}

// Routes
func introHandler(c *gin.Context) {
    c.HTML(200, "intro.html", nil)
}

func newAccount(c *gin.Context) {
    u := system.Create_user()
    c.HTML(200, "new_user.html", u)
}

func login(c *gin.Context) {
    username := strings.TrimSpace(c.PostForm("username"))
    password := strings.TrimSpace(c.PostForm("password"))
    session := sessions.Default(c)

    if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
        return
    }

    user, err := system.User_From_ID(username)
    if err != nil || !system.ValidatePassword(user, password) {
        sendInvalid(c)
        return
    }

    session.Set(userkey, user)
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
    user, err := validateUser(c); if err != nil {
        return
    }

    //check if the user has an unfinished quiz
    if user.Current_Quiz_Index < user.Current_Story_Index {
        fmt.Println("Moving user back to quiz")
        c.Redirect(http.StatusFound, "/private/quiz")
    }

    // check if they are done all the stories
    userStory, err := system.GetStory(user)
    if err != nil || system.Is_User_Complete(user) {
        c.HTML(200, "experimentComplete.html", nil)
        return
    }

    //TODO this needs a switch statement to determine the presentation type for the user's group

    SP := StoryPage {userStory.Name, user.User_ID, system.CommonWords, userStory.Story, userStory.Spans}
    c.HTML(200, "story.html", SP)
}

type storyEndPost struct {
    StartDate int
    EndDate int
    Wpm float32
}

func storyEndRoute(c *gin.Context) {
    user, err := validateUser(c); if err != nil {
        return
    }
    
    var record storyEndPost
    if err := c.ShouldBindJSON(&record); err != nil {
        fmt.Println("Error: ", err)
        c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
        return
    }
    system.Finish_Reading(user, record.StartDate, record.EndDate, record.Wpm)
    update_user(user,c)
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
    Date int
}

func actionRoute(c *gin.Context) {
    user, err := validateUser(c); if err != nil {
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
    Name string
}

func quizStartRoute(c *gin.Context) {
    user, err := validateUser(c); if err != nil {
        return
    }
    if user.Current_Quiz_Index == user.Current_Story_Index {
        fmt.Println("Moving user back to story")
        c.Redirect(http.StatusFound, "/private/story")
    }
    s, err := system.GetQuiz(user)
    if err != nil {
        return
    }

    quiz := new(QuizStruct)
    quiz.Questions = s.Questions
    quiz.Name = s.Name

    c.HTML(200, "quiz.html", &quiz)
}


type quizEndPost struct {
    StartDate int
    EndDate int
    Score int
}

func quizEndRoute(c *gin.Context) {
    user, err := validateUser(c); if err != nil {
        return
    }
    
    var record quizEndPost
    if err := c.ShouldBindJSON(&record); err != nil {
        fmt.Println("Error: ", err)
        c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
        return
    }
    fmt.Println("quiz record:",record)
    system.Finish_Quiz(user, record.StartDate, record.EndDate, record.Score)
    update_user(user, c)
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
    app.Static("/css","./css")
    app.Static("/scripts","./scripts")

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
        private.POST("/storyend",storyEndRoute)
        private.GET("/quiz", quizStartRoute)
        private.POST("/quizend", quizEndRoute)
        private.POST("/action", actionRoute)
    }
    app.Run(":"+PORT)
}

