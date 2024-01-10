/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"hanpack/cmd"
	"sync"
)

var wg sync.WaitGroup

func main() {
	ctx := context.Background()
	cmd.Execute(ctx)
}
