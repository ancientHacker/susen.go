package puzzle

import (
	"testing"
)

// Make sure error messages never panic and are never empty.  The
// testing of individual cases (and removal of unused errors) we
// leave to the functional testing done of other files.
func TestErrorNoPanicNoEmpty(t *testing.T) {
	defer (func() {
		if e := recover(); e != nil {
			t.Fatalf("Panic during testing: %v", e)
		}
	})()
	for sc := int(UnknownScope); sc <= int(MaxScope); sc++ {
		for st := int(UnknownStructure); st < int(MaxStructure); st++ {
			for at := int(UnknownAttribute); at < int(MaxAttribute); at++ {
				for co := int(UnknownCondition); co < int(MaxCondition); co++ {
					e := Error{
						Scope:     ErrorScope(sc),
						Structure: ErrorStructure(st),
						Attribute: ErrorAttribute(at),
						Condition: ErrorCondition(co),
					}
					m := e.Error()
					t.Log(m)
					if len(m) == 0 {
						t.Errorf("Empty error message for %+v", e)
					}
				}
			}
		}
	}
}
