package data

// Contains all of the setup and control of the storage. 
// Owns structs that are in storage:
//     USER
import (
    "fmt"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/Pallinder/go-randomdata"
    "log"
    "strings"
    "github.com/qubies/speed_reader/stories"
)

const DB_FILE = "./data/focused_reader.db"
const NUM_GROUPS = 3

// the generic user representation
type User struct {
    User_ID string
    Password string
    Group int
    Current_Story_Index int
}

type System struct {
    Database *sql.DB
    Group_Generator <-chan int
    Aborts []chan struct{}
    Stories []stories.Story
}

func Build_System(database_location, story_location string) *System {
    S := new(System)
    S.Database = create_db(database_location)
    S.Aborts = make([]chan struct{},0)
    S.Aborts = append(S.Aborts, make(chan struct{}))
    S.Group_Generator = generate_group(S.Aborts[0], NUM_GROUPS)
    S.Stories = stories.Load_Stories(story_location)

    return S
}

func create_db(location string) *sql.DB{
    db, err := sql.Open("sqlite3",location)
    if err != nil {
        log.Fatal(err)
    }

    schema := `
    PRAGMA foreign_keys = ON;
    create table IF NOT EXISTS Groups (Group_ID text not null primary key); 
    create table IF NOT EXISTS Users (User_ID text not null primary key, Password text, Group_ID text, FOREIGN KEY(Group_ID) REFERENCES Groups(Group_ID));
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

    new_user := &User{user_id, password, S.choose_group(), 0}
    tx, err := S.Database.Begin()
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

func (S *System) Validate_User(U *User) bool {
    var count int
    rows, err := S.Database.Query(fmt.Sprintf("select count(*) from Users where User_ID='%s' and Password='%s';",U.User_ID, U.Password))
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

