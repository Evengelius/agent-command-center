package confcheck

import (
	"encoding/csv"
	"os/exec"
	"strings"
)

// TasklistLister lists processes with `tasklist /fo csv /nh` (no extra
// dependency, no handle on other processes needed).
type TasklistLister struct{}

// Names returns the image name of each running process.
func (TasklistLister) Names() ([]string, error) {
	out, err := exec.Command("tasklist", "/fo", "csv", "/nh").Output()
	if err != nil {
		return nil, err
	}
	rows, err := csv.NewReader(strings.NewReader(string(out))).ReadAll()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, r := range rows {
		if len(r) > 0 {
			names = append(names, r[0])
		}
	}
	return names, nil
}
