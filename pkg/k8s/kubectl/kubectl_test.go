package kubectl

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/wzshiming/fake-k8s/pkg/utils"
)

func TestRun(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	err := Run(context.Background(), utils.IOStreams{
		In:     nil,
		Out:    buf,
		ErrOut: os.Stderr,
	}, "version", "--client")
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("no output")
	}
	t.Log(buf.String())
}
