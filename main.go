package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	host       string
	port       int
	configFile string
	id         int
	title      string
)

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Blog struct {
	Id        int    `json:"Id"`
	Title     string `json:"Title"`
	Content   string `json:"Content"`
	Category  string `json:"Category"`
	Tags      string `json:"Tags"`
	ViewCount int    `json:"ViewCount"`
	Author    string `json:"Author"`
}

func LoadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(file string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0644)
}

var rootCmd = &cobra.Command{
	Use:   "blog-cli",
	Short: "A simple CLI tool for managing blogs",
	Long:  `This tool allows you to manage your blogs easily through the command line.`,
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}

	configFile = filepath.Join(homeDir, "blog-cli.yaml")
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "Set the server and port",
		Run: func(cmd *cobra.Command, args []string) {
			config.Host = host
			config.Port = port
			err = SaveConfig(configFile, config)
			if err != nil {
				log.Fatalf("Error saving config: %v", err)
			}

			fmt.Printf("Configuration updated:\nHost: %s\nPort: %d\n", config.Host, config.Port)
		},
	}

	setCmd.Flags().StringVarP(&host, "server", "s", config.Host, "Host address")
	setCmd.Flags().IntVarP(&port, "port", "p", config.Port, "Port number")

	rootCmd.AddCommand(setCmd)

	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get the current server and port configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if err != nil {
				log.Fatalf("Error loading config: %v", err)
			}
			fmt.Printf("Current configuration:\nHost: %s\nPort: %d\n", config.Host, config.Port)
		},
	}
	rootCmd.AddCommand(getCmd)

	var pingCmd = &cobra.Command{
		Use:   "ping",
		Short: "Ping the configured host",
		Run: func(cmd *cobra.Command, args []string) {
			if err != nil {
				log.Fatalf("Error loading config: %v", err)
			}
			url := fmt.Sprintf("http://%s:%d/ping", config.Host, config.Port)
			fmt.Printf("Pinging %s...\n", url)
			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := client.Get(url)
			if err != nil {
				log.Fatalf("Failed to ping: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Ping successful: %s\n", string(body))
			} else {
				fmt.Printf("Ping failed with status code: %d\n", resp.StatusCode)
			}
		},
	}
	rootCmd.AddCommand(pingCmd)

	var addCmd = &cobra.Command{
		Use:   "add [filename]",
		Short: "Add a new blog post from a markdown file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fileName := args[0]
			content, err := os.ReadFile(fileName)
			if err != nil {
				fmt.Printf("Failed to read file: %s\n", err)
				return
			}
			fmt.Printf("Adding blog post from file: %s\n", fileName)

			blog := Blog{
				Id:        id,
				Title:     title,
				Content:   string(content),
				Category:  "",
				Tags:      "",
				ViewCount: 0,
				Author:    "",
			}

			jsonData, err := json.Marshal(blog)
			if err != nil {
				fmt.Printf("Failed to marshal blog post data: %s\n", err)
				return
			}

			url := fmt.Sprintf("http://%s:%d/blogs/updateOrAdd", config.Host, config.Port)

			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Printf("Failed to send POST request: %s\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Failed to read response body: %s\n", err)
				return
			}

			fmt.Printf("Response from server: %s\n", string(body))
		},
	}

	addCmd.Flags().IntVarP(&id, "id", "i", 0, "blog id")
	addCmd.Flags().StringVarP(&title, "title", "t", "", "blog title")

	rootCmd.AddCommand(addCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
