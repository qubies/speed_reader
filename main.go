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

type User struct {
    User_ID string
    Password string
    Group string
}

func (SP *StoryPage) handle_request(c *gin.Context) {
    c.HTML(200, "story.html", SP)
    session := sessions.Default(c)
	user := session.Get(userkey)
    fmt.Println("User: ", user)

}


func introHandler(c *gin.Context) {
    c.HTML(200, "intro.html", nil)
}

func quizHandler(c *gin.Context) {
    c.HTML(200, "quiz.html", nil)
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

func get_user_info(user string) (*User, bool) {
    if user_exists(user) {
        return new(User), false
    }
    rows, err := db.Query(fmt.Sprintf("select User_ID, Password, Group_ID from Users where User_ID='%s';",user))

    if err != nil {
        log.Fatal("Unable to query users: ", err)
    }
    defer rows.Close()
    
    u := new(User)
    err = rows.Scan(&u.User_ID, &u.Password, &u.Group)
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
    create table IF NOT EXISTS Stories (Story_ID integer primary key autoincrement, Date integer not null, wpm REAL, User_ID text, FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
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
    // templates = template.Must(template.ParseFiles("pages/quiz.html", "pages/intro.html", "pages/story.html", "pages/new_user.html"))
    templates = template.Must(template.ParseGlob("pages/*"))
}

func main() {
	defer db.Close()
    s := StoryPage {"This old man", "user", common_words, [][]string{{"l1","one,"}, {"l2","two."}, {"l3","three"}, {"four"}, {"five"}, {"six"}}, [][]int{{1,0,100}}}
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
    app.GET("/quiz", quizHandler)
    private := app.Group("/private")
    private.Use(AuthRequired) 
    {
        private.GET("/story", s.handle_request)
    }
    app.Run(":"+PORT)
}

