package state

import (
	"bytes"
	"fmt"
	"strconv"
)

func marshalRegistry(registry Registry) []byte {
	var b bytes.Buffer
	if registry.Default != "" {
		b.WriteString("default = ")
		b.WriteString(strconv.Quote(registry.Default))
		b.WriteString("\n")
		if len(registry.Sources) > 0 {
			b.WriteString("\n")
		}
	}

	for i, source := range registry.Sources {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[[source]]\n")
		b.WriteString("id = ")
		b.WriteString(strconv.Quote(source.ID))
		b.WriteString("\n")
		b.WriteString("path = ")
		b.WriteString(strconv.Quote(source.Path))
		b.WriteString("\n")
		fmt.Fprintf(&b, "enabled = %t\n", source.Enabled)
	}

	return b.Bytes()
}
