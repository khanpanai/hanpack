/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var rootCmd = &cobra.Command{
	Use:   "hanpack",
	Short: "Packing tool (initially for moodle plugins)",
	Long: `CLI tool for recursively packing folders skipping odd directory, such as .git, .idea, node_modules, etc.

Be aware, it can take more time if you run it in large folders`,
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlag("folder", cmd.PersistentFlags().Lookup("folder"))
		if err != nil {
			panic(err)
		}
		folder := viper.GetString("folder")

		var name string

		if folder == "." {
			getwd, err := os.Getwd()
			if err != nil {
				panic(err)
			}

			name = filepath.Base(getwd)
		} else {
			name = filepath.Base(folder)
		}

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

		fmt.Println(fmt.Sprintf("Packed: %s", path))
	}
}

func walk(paths chan<- string) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if path == "." {
			return nil
		}

		restrictedDir := append(viper.GetStringSlice("black_list"), viper.GetString("archive"))

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

	homedir, err := os.UserHomeDir()

	if err != nil {
		panic(err)
	}

	homedir = filepath.Join(homedir, ".hanpack")
	configPath := filepath.Join(homedir, "config.toml")
	if _, err = os.Stat(homedir); errors.Is(err, os.ErrNotExist) {

		err = os.Mkdir(homedir, os.ModePerm)

		if err != nil {
			panic("Can't create .hanpack folder")
		}

	}

	if _, err = os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(configPath)
		if err != nil {
			panic(errors.New("can't create cli config file to dump"))
		}

		var conf interface{}
		_ = toml.Unmarshal([]byte(`black_list = ["node_modules", ".git", "vendor", ".idea"]`), &conf)
		enc := toml.NewEncoder(file)
		_ = enc.Encode(conf)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(homedir)
	err = viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("Skipping these directories/files: %s", viper.GetStringSlice("black_list")))

	rootCmd.PersistentFlags().StringP("folder", "f", ".", "Choose directory to pack")
	viper.SetDefault("folder", ".")

}
