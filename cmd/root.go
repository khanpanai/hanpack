/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hanpack",
	Short: "Packing tool (initially for moodle plugins)",
	Long: `CLI tool for recursively packing folders skipping odd directory, such as .git, .idea, node_modules, etc.

Be aware, it can take more time if you run it in large folders`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		f := viper.GetString("folder")

		entries := make(chan string)

		go processEntries(entries)

		err := filepath.Walk(f, walk(entries))

		if err != nil {
			return
		}

		wg := &sync.WaitGroup{}

		go func() {
			wg.Wait()
			close(entries)
		}()

	},
}

func zip(wg *sync.WaitGroup, s string) {
	fmt.Println(s)
	wg.Done()
}

func processEntries(ch <-chan string) {
	for {
		select {
		case entry, ok := <-ch:
			if !ok {
				break
			}
			fmt.Println(entry)
		}
	}
}

func walk(ch chan<- string) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if path == "." {
			return nil
		}

		restrictedDir := []string{"node_modules", ".git", "vendor", ".idea"}

		for i := range restrictedDir {
			if restrictedDir[i] == path {
				return filepath.SkipDir
			}
		}

		if info.IsDir() {
			return nil
		}

		ch <- path

		return nil
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("folder", "f", ".", "Choose directory to pack")

	err := viper.BindPFlag("folder", rootCmd.PersistentFlags().Lookup("folder"))
	if err != nil {
		panic(err)
	}
	viper.SetDefault("folder", ".")
}
