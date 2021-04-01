//+build sqlite all

package nervatura

import _ "modernc.org/sqlite" //sqlite driver

func init() {
	registerDriver("sqlite")
}
