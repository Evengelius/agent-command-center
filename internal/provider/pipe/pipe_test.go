package pipe

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	br := bufio.NewReaderSize(strings.NewReader("ab\r\ncdefgh\nij"), 16)
	for _, want := range []struct {
		l    string
		size int
		over bool
	}{{"ab", 4, false}, {"", 7, true}, {"ij", 2, false}} {
		l, size, over, _ := ReadLine(br, 4)
		if string(l) != want.l || size != want.size || over != want.over {
			t.Errorf("got %q %d %v, want %+v", l, size, over, want)
		}
	}
}
