package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseID(t *testing.T) {
	type testCase struct {
		name  string
		idVal string
		isErr bool
	}

	tc := []testCase{
		{
			name:  "#1 no errors",
			idVal: "123e4567-e89b-12d3-a456-426655440000",
			isErr: false,
		},
		{
			name:  "#2 error",
			idVal: "1",
			isErr: true,
		},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			idRes, err := ParseID(test.idVal)
			if !assert.Equal(t, test.isErr, err != nil) {
				t.Errorf("error not compare [%v]!=[%v]", test.isErr, err)
				return
			}
			if test.isErr {
				return
			}

			if !assert.Equal(t, test.idVal, idRes.String()) {
				t.Errorf("res [%v]!=[%v]", test.idVal, idRes.String())
			}
		})
	}
}

func TestParse(t *testing.T) {
	type testCase struct {
		idStr   string
		login   string
		hash    string
		resUser User
		err     error
	}

}
