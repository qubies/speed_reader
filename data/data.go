package data

// Contains all of the setup and control of the storage. 
// Owns structs that are in storage:
//     USER
//     SYSTEM
// Depends on Stories as they are returned by the system
// interactions are done through system NOT USER.
// User is exported only for storage in session
import (
    "fmt"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/qubies/go-randomdata"
    "log"
    "strings"
    "github.com/qubies/speed_reader/stories"
	"os"
	"encoding/json"
	"errors"
	"io/ioutil"
)


// the generic user representation
type User struct {
    User_ID string
    password string
    Group int
    Current_Story_Index int
    Current_Quiz_Index int
}

func (U *User) get_story_id() int {
	return U.Current_Story_Index
}

// a user can read the story, and complete the quiz
func (U *User) hasReadStory() bool {
	return U.Current_Story_Index > U.Current_Quiz_Index
}

func (U *User) completeReading(){
	U.Current_Story_Index += 1
}

// advances to the next story
func (U *User) completeQuiz() error {
	
	if !U.hasReadStory() {
		return errors.New("user attempted quiz before story was read")
	}
	U.Current_Quiz_Index += 1
	return nil
}

type System struct {
    Database *sql.DB
    Group_Generator <-chan int
    Aborts []chan struct{}
    Stories []stories.Story
	CommonWords []string
}

func load_common_words(filename string) []string {
    common_word_file, err := os.Open(filename)
    var common_words []string
    if err != nil {
        log.Fatal("Common words file not found")
    }
    defer common_word_file.Close()
    byteValue, _ := ioutil.ReadAll(common_word_file)
    json.Unmarshal(byteValue, &common_words)
    return common_words
}

func Build_System(database_location, story_location, wordfile_location string, number_of_groups int) *System {
    S := new(System)
    S.Database = create_db(database_location)
    S.Aborts = make([]chan struct{},0)
    S.Aborts = append(S.Aborts, make(chan struct{}))
    S.Group_Generator = generate_group(S.Aborts[0], number_of_groups)
    S.Stories = stories.Load_Stories(story_location)
	S.CommonWords = load_common_words(wordfile_location)

    return S
}

func create_db(location string) *sql.DB{
    db, err := sql.Open("sqlite3",location)
    if err != nil {
        log.Fatal(err)
    }

    schema := `
    PRAGMA foreign_keys = ON;
    create table IF NOT EXISTS Users (User_ID text not null primary key, password text, Group_ID integer not null, Current_Story_Index integer default 0, Current_Quiz_Index integer default 0);
    create table IF NOT EXISTS Stories (Story_ID integer primary key autoincrement, Date integer not null, wpm REAL, Story_Name text, User_ID text, Score REAL,FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
    create table IF NOT EXISTS StoryActions (Action_ID integer primary key autoincrement, Date integer not null, Story_ID integer, Action integer not null, User_ID text not null, FOREIGN KEY(Story_ID) REFERENCES Stories(Story_ID), FOREIGN KEY(User_ID) REFERENCES Users(User_ID));
    `
    _, err = db.Exec(schema)
    if err != nil {
        log.Fatal("Unable to create DB: ", err)
    }
    sqlStmt := "INSERT INTO Groups(Group_ID) Values ('Experimental'), ('Control');"
    db.Exec(sqlStmt)
    return db
}

func generate_group(abort <-chan struct{}, max int) <-chan int {
    ch := make(chan int,10) // the generator is buffered becuase we never want to wait on it
    go func() {
        i := 0
        defer close(ch)
        for {
            select {
                case ch <- i:
                case <-abort: 
                    return
            }
            i += 1
            i %= max
        }
    }()
    return ch
}


func generate_user_id_and_password() (string, string) {
    return strings.ReplaceAll(randomdata.FullName(randomdata.RandomGender), " ", "_"), strings.ReplaceAll(randomdata.SillyName(), " ", "_")
}

func (S* System) User_exists(user string) (bool) {
    var count int
    rows, err := S.Database.Query(fmt.Sprintf("select count(*) from Users where User_ID='%s';",user))
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

func (S *System) choose_group() int {
    return <-S.Group_Generator
}

func (S* System) Create_user() *User{
    user_id := "" 
    password := ""
    for user_id == "" || S.User_exists(user_id) {
        user_id, password = generate_user_id_and_password()
    }

    new_user := &User{user_id, password, S.choose_group(), 0, 0}
    tx, err := S.Database.Begin()
    if err != nil {
        log.Fatal(err)
    }
    stmt, err := tx.Prepare(`INSERT into Users(User_ID, password, Group_ID) values (?,?,?)`) 
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()
    _, err = stmt.Exec(new_user.User_ID, new_user.password, new_user.Group)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("Added New User: ", new_user)
    tx.Commit()
    return new_user
}

func (S *System) Validate_User(U *User) bool {
    var count int
    rows, err := S.Database.Query(fmt.Sprintf("select count(*) from Users where User_ID='%s' and password='%s';",U.User_ID, U.password))
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

func (S *System) GetStory(U *User) (*stories.Story, error){
	if U.get_story_id() < len(S.Stories) {
		return &S.Stories[U.get_story_id()], nil
	}
	return nil, errors.New("No Stories Left")
}

func (S *System) User_Complete(U *User) bool {
	return U.get_story_id() >= len(S.Stories)
}

func (S *System) Complete_Reading_For(U *User) {
	U.completeReading()
}

func (S *System) User_From_ID(user_id string) (*User, error) {
    if !S.User_exists(user_id) {
        return nil, errors.New("User does not exist")
    }
    rows, err := S.Database.Query(fmt.Sprintf("select User_ID, password, Group_ID, Current_Story_Index, Current_Quiz_Index from Users where User_ID='%s' limit 1;",user_id))

    if err != nil {
		return nil, err
    }
    defer rows.Close()

    u := new(User)
    for rows.Next() {
        err = rows.Scan(&u.User_ID, &u.password, &u.Group, &u.Current_Story_Index, &u.Current_Quiz_Index)
    }
    return u, nil
}
