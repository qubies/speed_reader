package main

import (
    "log"
    // "io/ioutil"
    "net/http"
    "html/template"
    "os"
    "github.com/joho/godotenv"
)


type StoryPage struct {
    Title string
    Story [][]string
    Spans [][]int
}

type Page struct {

}

func introHandler(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("pages/intro.html")
    p := Page {}
    t.Execute(w,p)
}

func storyHandler(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("pages/story.html")
    s := StoryPage {"This old man", [][]string{{"l1","one"}, {"l2","two"}, {"l3","three"}, {"four"}, {"five"}, {"six"}}, [][]int{{1,2}}}
    t.Execute(w,s)
}

func init() {
    // loads values from .env into the system
    if err := godotenv.Load(); err != nil {
        log.Print("No .env file found")
    }
}

func main() {
    PORT := os.Getenv("PORT")
    if PORT == "" {
        PORT = "80"
    }
    http.HandleFunc("/", introHandler)
    http.HandleFunc("/story/", storyHandler)
    http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
    http.Handle("/scripts/", http.StripPrefix("/scripts/", http.FileServer(http.Dir("scripts"))))

    log.Fatal(http.ListenAndServe(":"+PORT, nil))
}

