package main

import (
    "errors"
    "strings"
    "log"
    // "io/ioutil"
    "html/template"
    "os"
    "github.com/joho/godotenv"
    // "encoding/json"
    "github.com/gin-gonic/gin"

    "fmt"
    // "math/rand"
    "net/http"
    "github.com/gin-gonic/contrib/sessions"
    data "github.com/qubies/speed_reader/data"
    "encoding/gob"
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
    startDate int
    endDate int
    wpm float32
}

func storyEndRoute(c *gin.Context) {
    user, err := validateUser(c); if err != nil {
        return
    }
    
    record := new(storyEndPost)
    if err := c.ShouldBindJSON(record); err != nil {
        fmt.Println("Error: ", err)
        c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
        return
    }
    system.Finish_Reading(user, record.startDate, record.endDate, record.wpm)
    fmt.Println(record)
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


// func quizStartRoute(c *gin.Context) {
//     session := sessions.Default(c)
//     story := session.Get("story").(string)


//     jsonFile, _ := os.Open("data/"+story)
//     defer jsonFile.Close()
//     byteValue, _ := ioutil.ReadAll(jsonFile)

//     var result StoryJson
//     json.Unmarshal([]byte(byteValue), &result)

//     var q_list []*Question
//     var correct_answer string
//     for _,q := range result.Questions {
//     var wrong_list []string
//         list :=[]string {"a", "b", "c", "d"}
//     options :=[]string {q.A, q.B, q.C, q.D}
//         for i,o := range list {
//             if o != q.Answer {
//                 wrong_list = append(wrong_list, options[i])
//             } else {
//                 correct_answer = options[i]
//             }
//         }
//         new_q :=new_question(q.Q_text, correct_answer, wrong_list)
//         q_list = append(q_list, new_q)
//     }
//     quiz:= Quiz{"", q_list}
//     c.HTML(200, "quiz.html", &quiz)
// }




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
        // private.GET("/quiz", quizStartRoute)
        // private.GET("/quizend", quizEndRoute)
        private.POST("/action", actionRoute)
    }
    app.Run(":"+PORT)
}

