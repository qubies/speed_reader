package data
import (
	"testing"
	"os"
	"reflect"
)

func TestGenerator(t *testing.T) {
    ch := generate_group(make(<-chan struct{}), 9)
    for x := 0; x<9; x++  {
        n := <-ch
        if x != n {
            t.Error("Got:", n, "Expected:", x)
        }
    }
    n := <-ch
    if n != 0 {
            t.Error("modulus wrap error... Got:", n, "Expected:", 0)
    }
}

func TestSystem(t *testing.T) {
	db_file := "./test.db"
	story_dir := "../stories/stories"
	wordfile_location := "../data/common_words.json"
	os.Remove(db_file) 

	//test user creation
	number_of_groups := 3
	data_system := Build_System(db_file, story_dir, wordfile_location, number_of_groups)
	for x:=0; x<=number_of_groups; x++ {
		test_user := data_system.Create_user()
		if test_user.Group != x && x < number_of_groups{
			t.Error("Incorrect Group ID: ", test_user)
		} else if test_user.Group != 0 && x == number_of_groups {
			t.Error("Incorrect Group ID, expected 0: ", test_user.Group)
		}
		
		if (!data_system.User_exists(test_user.User_ID)) {
			t.Error("User not verified in system...", test_user)
		}
	}
	if data_system.User_exists("thisisafakeuser") {
		t.Error("System acknowledges fake user")
	}

	// Test user vaildation
	new_user := data_system.Create_user()
	if !data_system.Validate_User(new_user) {
		t.Error("Unable to validate created user", new_user)
	}
	new_user.Password = "r"
	if data_system.Validate_User(new_user) {
		t.Error("Validated with incorrect password")
	}

	// Test common words is not null
	if reflect.DeepEqual(data_system.CommonWords, []string{}) {
		t.Error("Common words are empty")
	}


	// Test User Progression
	user := data_system.Create_user()
	if user.Has_Read_Story() || user.get_story_id() != 0 {
		t.Error("New User has read first story")
	}
	
	user.Complete_Reading()
	if !user.Has_Read_Story() || user.get_story_id() != 0 {
		t.Error("User should have read story")
	}

	user.Complete_Quiz()
	if user.Has_Read_Story() || user.get_story_id() != 1 {
		t.Error("User was not progressed to the next story")
	}

}
