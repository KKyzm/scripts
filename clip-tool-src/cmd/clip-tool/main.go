package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cliptool/internal/cliptool"
)

func main() {
	os.Exit(run())
}

func run() int {
	result, err := cliptool.Run(context.Background())
	if err != nil {
		if errors.Is(err, cliptool.ErrUserAborted) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if result.Notice != "" {
		fmt.Fprintln(os.Stderr, result.Notice)
	}
	if result.Stdout != "" {
		fmt.Fprint(os.Stdout, result.Stdout)
	}

	return 0
}
