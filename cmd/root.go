/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type entry struct {
	name string
	rc   io.ReadCloser
}

var rootCmd = &cobra.Command{
	Use:   "hanpack",
	Short: "Packing tool (initially for moodle plugins)",
	Long: `CLI tool for recursively packing folders skipping odd directory, such as .git, .idea, node_modules, etc.

Be aware, it can take more time if you run it in large folders`,
	PreRun: func(cmd *cobra.Command, args []string) {
		getwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		name := filepath.Base(getwd)

		viper.Set("archive", fmt.Sprintf("%s.zip", name))
	},
	Run: func(cmd *cobra.Command, args []string) {
		f := viper.GetString("folder")

		paths := make(chan string, 10)
		wg := sync.WaitGroup{}

		wg.Add(1)
		go processEntries(&wg, paths)

		err := filepath.Walk(f, walk(paths))

		if err != nil {
			return
		}
		go func() {
			close(paths)
		}()
		wg.Wait()
	},
}

func processEntries(wg *sync.WaitGroup, paths <-chan string) {
	defer wg.Done()

	f, err := os.Create(viper.GetString("archive"))

	if err != nil {
		log.Fatal(err)
	}

	zw := zip.NewWriter(f)

	defer func() {
		err := f.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}()

	defer func() {
		err = zw.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}()

	for path := range paths {
		file, err := os.Open(path)

		if err != nil {
			fmt.Println(path)
			log.Fatal(err)
		}

		create, err := zw.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.Copy(create, file)
		if err != nil {
			log.Fatal(err)
			return
		}
		if err != nil {
			panic(err)
		}
	}

}

func walk(paths chan<- string) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if path == "." {
			return nil
		}

		restrictedDir := []string{"node_modules", ".git", "vendor", ".idea", viper.GetString("archive")}

		for i := range restrictedDir {

			if restrictedDir[i] == filepath.Base(path) {
				return filepath.SkipDir
			}
		}

		if info.IsDir() {
			return nil
		}

		paths <- path
		return nil
	}
}

func Execute(ctx context.Context) {
	err := rootCmd.ExecuteContext(ctx)
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
