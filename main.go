package main

import (
    "strings"
    "log"
    "io/ioutil"
    "html/template"
    "os"
    "github.com/joho/godotenv"
    "encoding/json"
    "github.com/gin-gonic/gin"

    "fmt"
    "math/rand"
    "net/http"
    "time"
    "github.com/gin-gonic/contrib/sessions"
    db "github.com/qubies/speed_reader/database"
)
const (
    userkey = "user"
)


// globals 
var common_words []string
var templates *template.Template
var system *db.System


type StoryPage struct {
    Title string
    User string
    CommonWords []string
    Story [][]string
    Spans [][]int
}



type Choice struct {
    Correct string   `json:"correct"`
    Wrong   []string `json:"wrong"`
}

type Question struct {
    QuestionString string `json:"question_string"`
    Choices Choice `json:"choices"`
}

func new_question(question string, correct_answer string, wrong_answer []string) *Question {
    return &Question{question, Choice{correct_answer, wrong_answer}}
}

type Quiz struct {
    Name string
    Questions []*Question
}

type Quiz_Results struct {
    Date int
    Score float64
    Wpm float64
}

type Record struct {
    User_ID string
    Story_Name string
    Results *Quiz_Results
}

type Record_Update struct {
    Action int
    Date int
}

func handle_request(c *gin.Context) {

    session := sessions.Default(c)
    name := session.Get(userkey).(string)
    user, _ := get_user_info(name)

    // check if they are done all the stories
    if id < 0 {
        fmt.Println("all done stories")
        c.HTML(200, "all_done.html", nil)
        return
    }

    session.Set("story", files[id]) 
    session.Set("story_id", id)
    if err := session.Save(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
        return
    }


    if user.Group == "Control" {
        result.Spans = [][]int{}
        fmt.Println("Control")
    }

    SP := StoryPage {files[id], "user", common_words, result.Story, result.Spans}
    c.HTML(200, "story.html", SP)

}

func finish_story(c *gin.Context) {
    session := sessions.Default(c)
    name := session.Get(userkey).(string)
    story := session.Get("story").(string)
    fmt.Println(name, story, "in thing")
    record := Record{name, story, new(Quiz_Results)}
    if err := c.ShouldBindJSON(record.Results); err != nil {
        fmt.Println("Error: ", err)
        c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
        return
    }
    fmt.Println(record)
    add_record(&record)
}

func story_update(data *Record_Update) {

    //insert Record into dbStoryActions (Action_ID integer primary key autoincrement, Date integer not null, Story_ID integer, Action integer not null, User_ID text not null, FOREIGN KEY(Story_ID) REFERENCES Stories(Story_ID), FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
    sqlStmt := "INSERT INTO  StoryActions(Date,Story_ID, User_ID, Score) Values ($1, $2, $3, $4, $5);"
    fmt.Println(sqlStmt)
    _, err := db.Exec(sqlStmt, record.Results.Date, record.Results.Wpm, record.Story_Name, record.User_ID, record.Results.Score)
    if err != nil {
        fmt.Println("Error encountered in adding record: '", err, "'")
    }
}

func update_story(c *gin.Context) {
    // collect session vars
    session := sessions.Default(c)
    name := session.Get(userkey).(string)
    story := session.Get("story").(string)

    // collect post data
    var data Record_Update
    if err := c.ShouldBindJSON(&data); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
        return
    }
    fmt.Println("updated", data, name, story)
}

func introHandler(c *gin.Context) {
    c.HTML(200, "intro.html", nil)
}

func quizHandler(c *gin.Context) {
    session := sessions.Default(c)
    story := session.Get("story").(string)


    jsonFile, _ := os.Open("data/"+story)
    defer jsonFile.Close()
    byteValue, _ := ioutil.ReadAll(jsonFile)

    var result StoryJson
    json.Unmarshal([]byte(byteValue), &result)

    var q_list []*Question
    var correct_answer string
    for _,q := range result.Questions {
        var wrong_list []string
        list :=[]string {"a", "b", "c", "d"}
	options :=[]string {q.A, q.B, q.C, q.D}
        for i,o := range list {
            if o != q.Answer {
                wrong_list = append(wrong_list, options[i])
            } else {
                correct_answer = options[i]
            }
        }
        new_q :=new_question(q.Q_text, correct_answer, wrong_list)
        q_list = append(q_list, new_q)
    }
    quiz:= Quiz{"", q_list}
    c.HTML(200, "quiz.html", &quiz)
}

func new_handler(c *gin.Context) {
    u := add_user()
    c.HTML(200, "new_user.html", u)
}

func get_common_words() []string {
    common_word_file, err := os.Open("data/common_words.json")
    var common_words []string
    if err != nil {
        log.Fatal("Common words file not found")
    }
    defer common_word_file.Close()
    byteValue, _ := ioutil.ReadAll(common_word_file)
    json.Unmarshal(byteValue, &common_words)
    return common_words
}


func get_story_info(user string) ([]string) {
    rows, err := db.Query(fmt.Sprintf("select Story_Name from Stories where User_ID='%s';",user))
    if err != nil {
        log.Fatal("Unable to query stories: ", err)
    }
    defer rows.Close()
    var stories []string
    for rows.Next() {
        var str string
        err = rows.Scan(&str)
        stories = append(stories, str)
    }
    return stories
}

func add_record(record *Record) {

    //insert Record into db
    sqlStmt := "INSERT INTO stories (Date, wpm, Story_Name, User_ID, Score) Values ($1, $2, $3, $4, $5);"
    fmt.Println(sqlStmt)
    _, err := db.Exec(sqlStmt, record.Results.Date, record.Results.Wpm, record.Story_Name, record.User_ID, record.Results.Score)
    if err != nil {
        fmt.Println("Error encountered in adding record: '", err, "'")
    }
}

func get_user_info(user string) (*User, bool) {
    if !user_exists(user) {
        return new(User), false
    }
    rows, err := db.Query(fmt.Sprintf("select User_ID, Password, Group_ID from Users where User_ID='%s';",user))

    if err != nil {
        log.Fatal("Unable to query users: ", err)
    }
    defer rows.Close()

    u := new(User)

    for rows.Next() {
        err = rows.Scan(&u.User_ID, &u.Password, &u.Group)
    }
    return u, true
}


func login(c *gin.Context) {
    username := strings.TrimSpace(c.PostForm("username"))
    password := strings.TrimSpace(c.PostForm("password"))
    session := sessions.Default(c)

    if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
        return
    }

    U := new(User)
    U.User_ID = username
    U.Password = password
    U.Current_Story_Index = 0
    if U.Validate() {
        session.Set("user", U)
        if err := session.Save(); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
            return
        }
        c.Redirect(http.StatusFound, "/private/story")
    } else {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
        return
    }

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

// AuthRequired is a simple middleware to check the session
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
    // loads values from .env into the system
    rand.Seed(time.Now().Unix())
    create_db()
    common_words = get_common_words()
    if err := godotenv.Load(); err != nil {
        log.Print("No .env file found")
    }
    templates = template.Must(template.ParseGlob("pages/*"))

}

func main() {
    defer db.Close()
    PORT := os.Getenv("PORT")
    if PORT == "" {
        PORT = "80"
    }
    app := gin.Default()
    app.Use(sessions.Sessions("speed_reading", sessions.NewCookieStore([]byte("secret"))))
    app.SetHTMLTemplate(templates)

    //routing
    app.Static("/css","./css")
    app.Static("/scripts","./scripts")

    // so there currently aren't images, but if we want....
    // app.Static("/images","./images")

    app.GET("/", introHandler)
    app.GET("/new_account", new_handler)
    app.POST("/login", login)
    app.GET("/logout", logout)

    private := app.Group("/private")
    private.Use(AuthRequired) 
    {
        private.GET("/story", handle_request)
        private.GET("/quiz", quizHandler)
        private.POST("/record",finish_story)
        private.POST("/update_story", update_story)
    }
    app.Run(":"+PORT)
}

