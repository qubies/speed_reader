package data
import "testing"

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
	data_system := Build_System("./test.db", "../stories/stories")
}
