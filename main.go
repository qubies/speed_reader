package main

import (
    "strconv"
    "strings"
    "log"
    "io/ioutil"
    "html/template"
    "os"
    "github.com/joho/godotenv"
    "encoding/json"
    "github.com/gin-gonic/gin"

    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "database/sql"
    "github.com/Pallinder/go-randomdata"
    "math/rand"
    "net/http"
    "time"
    "github.com/gin-gonic/contrib/sessions"
)
const (
    userkey = "user"
)
// globals 
var common_words []string
var templates *template.Template
var db *sql.DB
var groups []string = []string{"Experimental", "Control"}
var last_group int = 1


type StoryPage struct {
    Title string
    User string
    CommonWords []string
    Story [][]string
    Spans [][]int
}

type JsonQuestion struct {
    Q_num string `json:"q_num"`
    Q_text string	`json:"q_text"`
    Answer string	`json:"answer"`
    A string	`json:"a."`
    B string	`json:"b."`
    C string	`json:"c."`
    D string	`json:"d." `    
}

type StoryJson struct {
    Story [][]string		`json:"story"`
    Spans [][]int		`json:"spans"`
    Questions []JsonQuestion	`json:"questions"`
}

type User struct {
    User_ID string
    Password string
    Group string
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
type Record struct {
    User_ID string
    Story_Name string
    Date int
    Wpm float64
    Record float64
}

func string_in_slice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func handle_request(c *gin.Context) {

    files := []string{"carnivorous-plants.json","hyperinflation.json","worst-game-ever.json", "black-friday.json", "google.json", "honey-badgers.json", "hummingbirds.json", "koko.json", "metal-detectors.json", "mongooses.json", "reading.json", "seat-belts.json", "trampolines.json"}

    session := sessions.Default(c)
    name := session.Get(userkey).(string)
    user, _ := get_user_info(name)

    stories := get_story_info(name)

    id := -1
    for i, name := range files {
        if string_in_slice(name, stories) {
            continue
        }
        id = i
    }

    if id < 0 {
        fmt.Println("all done stories")
        c.HTML(200, "all_done.html", nil)
        return
    }

    jsonFile, _ := os.Open("data/"+files[id])
    defer jsonFile.Close()
    byteValue, _ := ioutil.ReadAll(jsonFile)

    session.Set("story", files[id]) // In real world usage you'd set this to the users ID
    if err := session.Save(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
        return
    }

    var result StoryJson
    json.Unmarshal([]byte(byteValue), &result)

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
    wpm,_ := strconv.ParseFloat(c.Query("wpm"), 64)
    date,_ := strconv.Atoi(c.Query("date"))

    mark,_ := strconv.ParseFloat(c.Query("mark"), 64)
    record := Record{name, story, date, wpm, mark}

    fmt.Printf("%v %v %v %v", wpm, date, mark, name)
    add_record(&record)
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

func generate_user_info() *User {

    return &User{strings.ReplaceAll(randomdata.FullName(randomdata.RandomGender), " ", "_"), strings.ReplaceAll(randomdata.SillyName(), " ", "_"), choose_group()}

}

func choose_group() string {
    last_group = (last_group + 1) % len(groups)
    return groups[last_group]
    // return groups[rand.Intn(len(groups))]

}

func add_user() *User{
    new_user := generate_user_info()
    for user_exists(new_user.User_ID) {
        new_user = generate_user_info()
    }
    tx, err := db.Begin()
    if err != nil {
        log.Fatal(err)
    }
    stmt, err := tx.Prepare(`INSERT into Users(User_ID, Password, Group_ID) values (?,?,?)`) 
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()
    _, err = stmt.Exec(new_user.User_ID, new_user.Password, new_user.Group)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("Added New User: ", new_user)
    tx.Commit()
    return new_user
}

func user_exists(user string) (bool) {
    var count int
    rows, err := db.Query(fmt.Sprintf("select count(*) from Users where User_ID='%s';",user))
    if err != nil {
        log.Fatal("count query error: ", err)
    }

    for rows.Next() {
        err:= rows.Scan(&count)
        if err != nil {
            log.Fatal("ooopse")
        }
    }
    return count>0
}

func (U *User) Validate() bool {
    var count int
    rows, err := db.Query(fmt.Sprintf("select count(*) from Users where User_ID='%s' and Password='%s';",U.User_ID, U.Password))
    if err != nil {
        log.Fatal("count query error: ", err)
    }

    for rows.Next() {
        err:= rows.Scan(&count)
        if err != nil {
            log.Fatal("ooopse")
        }
    }
    return count>0
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

func add_record(record *Record) (bool){

    //insert Record into db
    sqlStmt := "INSERT INTO Stories (Date, wpm, Story_Name, User_ID, Record) Values ($1, $2, $3, $4, $5);"
    _, err := db.Exec(sqlStmt, record.Date, record.Wpm, record.Story_Name, record.User_ID, record.Record)
    fmt.Println(err)
    //retun true false success
    return false
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

func create_db() {
    err := *new(error)
    db, err = sql.Open("sqlite3", "./data/656_project.db")
    if err != nil {
        log.Fatal(err)
    }

    sqlStmt := `
    PRAGMA foreign_keys = ON;
    create table IF NOT EXISTS Groups (Group_ID text not null primary key); 
    create table IF NOT EXISTS Users (User_ID text not null primary key, Password text, Group_ID text, FOREIGN KEY(Group_ID) REFERENCES Groups(Group_ID));
    create table IF NOT EXISTS Stories (Story_ID integer primary key autoincrement, Date integer not null, wpm REAL, Story_Name text, User_ID text, Record REAL,FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
    `
    _, err = db.Exec(sqlStmt)
    if err != nil {
        log.Fatal("Unable to create DB: ", err)
    }
    sqlStmt = "INSERT INTO Groups(Group_ID) Values ('Experimental'), ('Control');"
    db.Exec(sqlStmt)
}

func login(c *gin.Context) {
    username := c.PostForm("username")
    password := c.PostForm("password")
    session := sessions.Default(c)

    if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
        return
    }

    U := new(User)
    U.User_ID = username
    U.Password = password
    if U.Validate() {
        session.Set(userkey, username) // In real world usage you'd set this to the users ID
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

    app.Static("/images","./images")
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
    }
    app.Run(":"+PORT)
}

