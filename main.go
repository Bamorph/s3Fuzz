package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	concurrency int
	delay       int
	wg          sync.WaitGroup
)

// buildNames generates a list of possible names based on keywords, mutations, suffixes, and prefixes.
func buildNames(keywords, mutations, suffixes, prefixes []string) []string {
	var names []string

	for _, keyword := range keywords {
		names = append(names, keyword)

		for _, mutation := range mutations {
			if mutation != "" {
				names = append(names, fmt.Sprintf("%s%s", keyword, mutation))
				names = append(names, fmt.Sprintf("%s.%s", keyword, mutation))
				names = append(names, fmt.Sprintf("%s-%s", keyword, mutation))
				names = append(names, fmt.Sprintf("%s%s", mutation, keyword))
				names = append(names, fmt.Sprintf("%s.%s", mutation, keyword))
				names = append(names, fmt.Sprintf("%s-%s", mutation, keyword))
			}
		}

		for _, suffix := range suffixes {
			if suffix != "" {
				names = append(names, fmt.Sprintf("%s-%s", keyword, suffix))
				names = append(names, fmt.Sprintf("%s.%s", keyword, suffix))
				names = append(names, fmt.Sprintf("%s%s", keyword, suffix))
			}

			for _, prefix := range prefixes {
				if prefix != "" {
					names = append(names, fmt.Sprintf("%s-%s", prefix, keyword))
					names = append(names, fmt.Sprintf("%s.%s", prefix, keyword))
					names = append(names, fmt.Sprintf("%s%s", prefix, keyword))
				}

				if prefix != "" && suffix != "" {
					names = append(names, fmt.Sprintf("%s-%s-%s", prefix, keyword, suffix))
					names = append(names, fmt.Sprintf("%s.%s.%s", prefix, keyword, suffix))
					names = append(names, fmt.Sprintf("%s%s%s", prefix, keyword, suffix))
				}
			}
		}
	}

	return names
}

// removeDuplicates removes duplicate strings from a slice.
func removeDuplicates(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, str := range input {
		if !seen[str] {
			result = append(result, str)
			seen[str] = true
		}
	}

	return result
}

// readLines reads lines from a file and returns them as a slice.
func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, cleanText(scanner.Text()))
	}

	return lines, nil
}

// cleanTextList applies cleanText to a list of strings.
func cleanTextList(textList []string) []string {
	var cleanedList []string

	for _, text := range textList {
		cleanedText := cleanText(text)
		cleanedList = append(cleanedList, cleanedText)
	}

	return cleanedList
}

// cleanText removes unwanted characters from a string.
func cleanText(text string) string {
	bannedChars := regexp.MustCompile(`[^a-z0-9.-]`)
	textLower := strings.ToLower(text)
	textClean := bannedChars.ReplaceAllString(textLower, "")

	return textClean
}

// appendAWS prepends "https://" and appends ".s3.amazonaws.com" to a string.
func appendAWS(name string) string {
	return "https://" + name + ".s3.amazonaws.com"
}

// Contents represents the Key field in XML content.
type Contents struct {
	Key string `xml:"Key"`
}

// ListBucketResult represents the XML structure of an S3 bucket.
type ListBucketResult struct {
	Contents []Contents `xml:"Contents"`
}

// readXMLContent reads and processes XML content from an HTTP response.
func readXMLContent(body io.Reader, bucket string) {
	xmlContent, err := ioutil.ReadAll(body)
	if err != nil {
		fmt.Println("Error reading XML content:", err)
		return
	}

	var result ListBucketResult
	err = xml.Unmarshal(xmlContent, &result)
	if err != nil {
		fmt.Println("Error Unmarshalling XML:", err)
		return
	}

	if len(result.Contents) == 0 {
		redPrint("EMPTY BUCKET")
	}

	for _, content := range result.Contents {
		fmt.Printf("%s/%s\n", bucket, content.Key)
	}
}

// resolveURL sends an HTTP request to the given URL and processes the response.
func resolveURL(url string) {
	defer wg.Done()

	time.Sleep(time.Duration(delay) * time.Millisecond)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		greenPrint("Open: " + url)

		if strings.Contains(resp.Header.Get("Content-Type"), "xml") {
			readXMLContent(resp.Body, url)
		}
	case http.StatusForbidden:
		yellowPrint("Protected: " + url)
	}
}

// cyanPrint prints text in cyan color.
func cyanPrint(text string) {
	fmt.Printf("\033[K")
	cyan := color.New(color.FgHiCyan, color.Bold)
	cyan.Println(text)
}

// redPrint prints text in red color.
func redPrint(text string) {
	fmt.Printf("\033[K")
	red := color.New(color.FgRed, color.Bold)
	red.Println(text)
}

// greenPrint prints text in green color.
func greenPrint(text string) {
	fmt.Printf("\033[K")
	green := color.New(color.FgGreen, color.Bold)
	green.Println(text)
}

// yellowPrint prints text in yellow color.
func yellowPrint(text string) {
	fmt.Printf("\033[K")
	yellow := color.New(color.FgHiYellow, color.Bold)
	yellow.Println(text)
}

// addWorker creates concurrent workers to resolve URLs.
func addWorker(nameCh <-chan string) {
	for name := range nameCh {
		url := appendAWS(name)
		resolveURL(url)
	}
}

// main is the entry point of the program.
func main() {
	var (
		prefixs   string
		suffixs   string
		mutations string
		keywords  []string
	)

	flag.StringVar(&prefixs, "p", "", "prefix file")
	flag.StringVar(&suffixs, "s", "", "suffix file")
	flag.StringVar(&mutations, "w", "", "wordlist file")

	flag.IntVar(&concurrency, "c", 5, "Number of concurrent workers")
	flag.IntVar(&delay, "d", 1000, "Delay time in milliseconds")

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("TODO: print help page if no keywords are supplied")
		os.Exit(1)
	}

	keywords = cleanTextList(flag.Args())

	prefixLines, err
