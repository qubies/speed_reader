package main

import (
    "log"
    "io/ioutil"
    "html/template"
    "os"
    "github.com/joho/godotenv"
    "encoding/json"
    "github.com/gin-gonic/gin"
    "fmt"
)

// globals 
var common_words []string
var templates *template.Template


type StoryPage struct {
    Title string
    User string
    CommonWords []string
    Story [][]string
    Spans [][]int
}


func (SP *StoryPage) handle_request(c *gin.Context) {
    c.HTML(200, "story.html", SP)
}


func introHandler(c *gin.Context) {
    c.HTML(200, "intro.html", nil)
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

// do all of the goodness setup stuffs
func init() {
    // loads values from .env into the system
    common_words = get_common_words()
    if err := godotenv.Load(); err != nil {
        log.Print("No .env file found")
    }
    templates = template.Must(template.ParseFiles("pages/quiz.html", "pages/intro.html", "pages/story.html"))
    fmt.Printf("%+v\n", templates)
}

func main() {
    s := StoryPage {"This old man", "user", common_words, [][]string{{"l1","one,"}, {"l2","two."}, {"l3","three"}, {"four"}, {"five"}, {"six"}}, [][]int{{1,2}}}
    PORT := os.Getenv("PORT")
    if PORT == "" {
        PORT = "80"
    }
    app := gin.Default()
    app.SetHTMLTemplate(templates)

    //routing
    app.Static("/css","./css")
    app.Static("/scripts","./scripts")
    app.GET("/story", s.handle_request)
    app.GET("/", introHandler)
    app.Run(":"+PORT)

    // http.HandleFunc("/", introHandler)
    // http.HandleFunc("/story/", s.handle_request)
    // http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
    // http.Handle("/scripts/", http.StripPrefix("/scripts/", http.FileServer(http.Dir("scripts"))))

    // log.Fatal(http.ListenAndServe(":"+PORT, nil))
}

