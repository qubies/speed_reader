package data

import (
	"fmt"
	"log"
	"os"
	"testing"
	// "reflect"
	// "math/rand"
)

func removeFile(name string) {
	e := os.Remove(name)
	if e != nil {
		log.Fatal(e)
	}
}

func TestLoadStories(t *testing.T) {
	stories := *load_stories()
	t.Log(stories.Data[0])
}

func TestGroupGenerator(t *testing.T) {
	t.Log("testing Groups")
	loadGroups()
}

func TestBuildSystem(t *testing.T) {
	testDB := "test_db.sql"
	Build_System(testDB, "./common_words.json")
	defer removeFile(testDB)
}

func TestUserFunctions(t *testing.T) {
	testDB := "test_db.sql"
	system := Build_System(testDB, "./common_words.json")
	defer removeFile(testDB)

	// make sure that the users are created correctly
	if len(system.Users) != USER_COUNT {
		t.Errorf("User Count incorrect, expected %d got %d", USER_COUNT, len(system.Users))
	}
	originalUsers := make(map[string]*Status)

	for _, u := range system.Users {
		originalUsers[u.User_ID] = system.GetCurrentEvent(u)
		pos, err := system.GetPosition(u)
		if err != nil {
			t.Error(err)
		}
		if pos != 0 {
			t.Error("user position is not 0 at creation")
		}
		if !system.User_exists(u.User_ID) {
			t.Error(fmt.Sprintf("validation failed for user '%s'", u.User_ID))
		}
		if system.User_exists("notauser") {
			t.Error("notauser exists!!! uh oh")
		}
	}

	// advance a user

	for _, u := range system.Users {
		err := system.AdvanceUser(u)
		if err != nil {
			t.Error("Advance failed: ", err)
		}
	}

	//rebuild and see if info stays put

	system = Build_System(testDB, "./common_words.json")
	for _, u := range system.Users {
		if _, ok := originalUsers[u.User_ID]; !ok {
			t.Error("User '", u.User_ID, "' Missing from reload")
			//do something here
		} else if u.position != 1 {
			t.Error("Position not saved after update expected 1, got ", u.position, " for user ", u.User_ID)
		}
		originalUsers[u.User_ID] = system.GetCurrentEvent(u) // this is needed to restore the Story addresses
	}

	//test to see if users advance Story correctly
	//settings should be the same as they were, except they should be on the quiz
	for x := 0; x < 4; x++ {

		for _, u := range system.Users {
			stat := system.GetCurrentEvent(u)
			prior := originalUsers[u.User_ID]
			prior.Event = "quiz"

			if *stat != *prior {
				t.Errorf("User state changed incorrectly, wanted: %+v got %+v on loop %d, %s, %s", prior, stat, x, prior.Story.Title, stat.Story.Title)
			}
			err := system.AdvanceUser(u)
			if err != nil {
				t.Error("Advance failed: ", err)
			}
		}

		for _, u := range system.Users {
			stat := system.GetCurrentEvent(u)
			prior := originalUsers[u.User_ID]
			prior.Event = "questionnaire"

			if *stat != *prior {
				t.Errorf("User state changed incorrectly, wanted: %+v got %+v on loop %d", prior, stat, x)
			}
			err := system.AdvanceUser(u)
			if err != nil {
				t.Error("Advance failed: ", err)
			}
		}

		for _, u := range system.Users {
			stat := system.GetCurrentEvent(u)
			prior := originalUsers[u.User_ID]
			prior.Event = "Story"

			if stat.storyIndex == prior.storyIndex {
				t.Errorf("Story index did not advance: %d got %d", prior.storyIndex, stat.storyIndex)
			}
			if stat.treatmentType == prior.treatmentType {
				t.Errorf("Treatment Type did not advance: %d got %d", prior.treatmentType, stat.treatmentType)
			}
			if stat.Story == prior.Story {
				t.Error("Story should have changed on final Event")
			}
			if x == 3 {
				if !stat.Completed {
					t.Error("Expected routine to finish at the end of loop 3...")
				}

			} else {
				if stat.Completed == true && x != 3 {
					t.Error("Finished too soon on loop ", x)
				}
				err := system.AdvanceUser(u)
				if err != nil {
					t.Error("Advance failed: ", err)
				}

			}
			originalUsers[u.User_ID] = stat
		}
		t.Log("stage ", x)
	}
}

// func TestGenerator(t *testing.T) {
//     ch := generate_group(make(<-chan struct{}), 9)
//     for x := 0; x<9; x++  {
//         n := <-ch
//         if x != n {
//             t.Error("Got:", n, "Expected:", x)
//         }
//     }
//     n := <-ch
//     if n != 0 {
//             t.Error("modulus wrap error... Got:", n, "Expected:", 0)
//     }
// }

// func TestSystem(t *testing.T) {
//     db_file := "./test.db"
//     story_dir := "../stories/test_folder"
//     wordfile_location := "../data/common_words.json"
//     os.Remove(db_file)

//     //test user creation
//     number_of_groups := 3
//     data_system := Build_System(db_file, story_dir, wordfile_location, number_of_groups)
//     for x:=0; x<=number_of_groups; x++ {
//         test_user := data_system.Create_user()
//         if test_user.Group != x && x < number_of_groups{
//             t.Error("Incorrect Group ID: ", test_user)
//         } else if test_user.Group != 0 && x == number_of_groups {
//             t.Error("Incorrect Group ID, expected 0: ", test_user.Group)
//         }

//         if (!data_system.User_exists(test_user.User_ID)) {
//             t.Error("User not verified in system...", test_user)
//         }
//     }
//     if data_system.User_exists("thisisafakeuser") {
//         t.Error("System acknowledges fake user")
//     }

//     // Test user vaildation
//     new_user := data_system.Create_user()
//     if !data_system.Validate_User(new_user) {
//         t.Error("Unable to validate created user", new_user)
//     }
//     new_user.Password = "r"
//     if data_system.Validate_User(new_user) {
//         t.Error("Validated with incorrect password")
//     }

//     // Test common words is not null
//     if reflect.DeepEqual(data_system.CommonWords, []string{}) {
//         t.Error("Common words are empty")
//     }

//     // Test User Progression
//     user := data_system.Create_user()
//     if user.hasReadStory() || user.get_story_id() != 0 {
//         t.Error("New User has read first Story")
//     }

//     if user.completeQuiz() == nil {
//         t.Error("User was allowed to complete quiz before reading")
//     }

//     // make sure it returns the first story_location
//     s, err := data_system.GetStory(user)
//     if err != nil {
//         t.Error("System returned an error", err)
//     }
//     if s.Name != "Beyonce" {
//         t.Error("first Story is wrong", s.Name)
//     }

//     // complete the reading
//     user.completeReading()

//     // the Story should be onto the next

//     s, err = data_system.GetStory(user)
//     if err != nil {
//         t.Error("System returned an error", err)
//     }
//     if s.Name != "Sino-Tibetan_relations_during_the_Ming_dynasty" {
//         t.Error("second Story is wrong", s.Name)
//     }

//     //but they should have read the Story

//     if !user.hasReadStory() || user.get_story_id() != 1 {
//         t.Error("User should have read Story")
//     }

//     // now that they complete the quiz...

//     user.completeQuiz()
//     if user.hasReadStory() || user.get_story_id() != 1 {
//         t.Error("User was not progressed to the next Story")
//     }

//     // they should move on to the next Story

//     s, err = data_system.GetStory(user)
//     if err != nil {
//         t.Error("System returned an error", err)
//     }
//     if s.Name != "Sino-Tibetan_relations_during_the_Ming_dynasty" {
//         t.Error("first Story is wrong", s.Name)
//     }

//     // because there are 2 stories, this should put the user at the end...
//     if data_system.Is_User_Complete(user) {
//         t.Error("System failed to recognize the user was not done")
//     }

//     user.completeReading()
//     user.completeQuiz()

//     if !data_system.Is_User_Complete(user) {
//         t.Error("System failed to recognize completion")
//     }
//     s, err = data_system.GetStory(user)
//     if err == nil {
//         t.Error("System should have reached the end")
//     }

//     // Test user creation system with loading....
//     for x:=0; x < 1000; x++ {
//         u := data_system.Create_user()

//         if !data_system.Validate_User(u) {
//             t.Error("unable to validate user...:", u)
//         }

//         u_2, _ := data_system.User_From_ID(u.User_ID)
//         if !reflect.DeepEqual(u, u_2) {
//             t.Error("System inequaility in returned user from generated user")
//         }
//     }

//     // this part shows how the API should be used....
//     u:= data_system.Create_user()

//     // record some actions as the user works.
//     for i := 0; i<10; i++ {
//         err := data_system.Record_Action(u, i, 0)
//         if err != nil {
//             t.Error("Error encoundered updating records", err)
//         }
//     }

//     // when the user finishes reading the text:
//     for !data_system.Is_User_Complete(u) {
//         err = data_system.Finish_Reading(u, 0,0,rand.Float32()*100)
//         if err != nil{
//             t.Error("Error returned from Finish marking: ", err)
//         }
//         if !u.hasReadStory(){
//             t.Error("Not finished Story after marking")
//         }

//         err = data_system.Finish_Quiz(u, 0,0,rand.Int())
//         if err != nil{
//             t.Error("Error returned from Finishing Quiz: ", err)
//         }
//         if u.hasReadStory() {
//             t.Error("Not finished quiz after marking")
//         }
//     }
// }
