package main

import (
    "log"
    "io/ioutil"
    "net/http"
    "html/template"
    "os"
    "github.com/joho/godotenv"
    "encoding/json"
)

// globals 
var common_words []string
var templates *template.Template


type StoryPage struct {
    Title string
    CommonWords []string
    Story [][]string
    Spans [][]int
}

func (SP *StoryPage) handle_request(w http.ResponseWriter, r *http.Request) {
    err := templates.ExecuteTemplate(w, "story.html", SP)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}


func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
}

type Page struct {

}

func introHandler(w http.ResponseWriter, r *http.Request) {
    p := Page {}
    err := templates.ExecuteTemplate(w, "intro.html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}


func init() {
    // loads values from .env into the system
    common_words = get_common_words()
    if err := godotenv.Load(); err != nil {
        log.Print("No .env file found")
    }
    templates = template.Must(template.ParseFiles("pages/quiz.html", "pages/intro.html", "pages/story.html"))
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

func main() {
    s := StoryPage {"This old man", common_words, [][]string{{"l1","one"}, {"l2","two"}, {"l3","three"}, {"four"}, {"five"}, {"six"}}, [][]int{{1,2}}}
    PORT := os.Getenv("PORT")
    if PORT == "" {
        PORT = "80"
    }

    http.HandleFunc("/", introHandler)
    http.HandleFunc("/story/", s.handle_request)
    http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
    http.Handle("/scripts/", http.StripPrefix("/scripts/", http.FileServer(http.Dir("scripts"))))

    log.Fatal(http.ListenAndServe(":"+PORT, nil))
}

