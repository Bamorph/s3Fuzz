package main

import (
	"bufio"
	"flag"
	"fmt"
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

func buildNames(keywords []string, mutations []string, suffixs []string, prefixs []string) []string {
	var names []string
	for _, keyword := range keywords {
		names = append(names, keyword)

		for _, mutation := range mutations {
			names = append(names, fmt.Sprintf("%s%s", keyword, mutation))
			names = append(names, fmt.Sprintf("%s.%s", keyword, mutation))
			names = append(names, fmt.Sprintf("%s-%s", keyword, mutation))
			names = append(names, fmt.Sprintf("%s%s", mutation, keyword))
			names = append(names, fmt.Sprintf("%s.%s", mutation, keyword))
			names = append(names, fmt.Sprintf("%s-%s", mutation, keyword))
		}

		for _, suffix := range suffixs {
			if suffix != "" {
				names = append(names, fmt.Sprintf("%s-%s", keyword, suffix))
				names = append(names, fmt.Sprintf("%s.%s", keyword, suffix))
				names = append(names, fmt.Sprintf("%s%s", keyword, suffix))
			}
			for _, prefix := range prefixs {
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

func cleanTextList(textList []string) []string {
	var cleanedList []string

	for _, text := range textList {
		cleanedText := cleanText(text)
		cleanedList = append(cleanedList, cleanedText)
	}

	return cleanedList
}

func cleanText(text string) string {

	bannedChars := regexp.MustCompile(`[^a-z0-9.-]`)
	textLower := strings.ToLower(text)
	textClean := bannedChars.ReplaceAllString(textLower, "")

	return textClean
}

func appendAWS(names []string) []string {
	var result []string

	for _, n := range names {
		result = append(result, "https://"+n+".s3.amazonaws.com")
	}

	return result
}

func resolveurl(url string) {
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
	case http.StatusForbidden:
		yellowPrint("Protected: " + url)
	case http.StatusNotFound:
		return
	}

}

func cyanPrint(text string) {
	fmt.Printf("\033[K")
	cyan := color.New(color.FgHiCyan, color.Bold)
	cyan.Println(text)
}
func greenPrint(text string) {
	fmt.Printf("\033[K")
	green := color.New(color.FgGreen, color.Bold)
	green.Println(text)
}
func yellowPrint(text string) {
	fmt.Printf("\033[K")
	yellow := color.New(color.FgHiYellow, color.Bold)
	yellow.Println(text)
}

// const version = "0.0.2"

func main() {
	var (
		prefixs   string
		suffixs   string
		mutations string
		keywords  []string
	)

	flag.StringVar(&prefixs, "p", "", "TODO: Description")
	flag.StringVar(&suffixs, "s", "", "TODO: Description")
	flag.StringVar(&mutations, "w", "", "wordlist file")

	flag.IntVar(&concurrency, "c", 5, "Number of concurrent workers")
	flag.IntVar(&delay, "d", 1000, "Delay time in milliseconds")

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("TODO: print help page if no keywords are supplied")
		os.Exit(1)
	}

	keywords = cleanTextList(flag.Args())

	prefixLines, err := readLines(prefixs)
	if err != nil {
		if os.IsNotExist(err) {
			prefixLines = []string{""}
		} else {
			fmt.Println("Error reading prefix file:", err)
			os.Exit(1)
		}
	}

	suffixLines, err := readLines(suffixs)
	if err != nil {
		if os.IsNotExist(err) {
			suffixLines = []string{""}
		} else {
			fmt.Println("Error reading suffix file:", err)
			os.Exit(1)
		}
	}
	mutationsLines, err := readLines(mutations)
	if err != nil {
		if os.IsNotExist(err) {
			mutationsLines = []string{""}
		} else {
			os.Exit(1)
		}
	}
	var names = buildNames(keywords, suffixLines, mutationsLines, prefixLines)

	names = removeDuplicates(names)

	urls := appendAWS(names)
	total_test := len(urls)

	fmt.Printf("[+] Keywords: %s\n", keywords)
	fmt.Printf("[+] Total urls to test: %v\n\n", total_test)
	cyanPrint("[+] Amazon S3 Buckets\n")

	urlCh := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			for url := range urlCh {
				resolveurl(url)
			}
		}()
	}
	for i, url := range urls {
		fmt.Printf("\033[K")
		// fmt.Printf("%d / %d, URL: %s\r", i, len(urls), url)
		fmt.Printf("%d / %d, URL: %s\r", i, len(urls), url[:20]) // only show the first 10 characters of the url
		// fmt.Printf("%d / %d\r", i, len(urls))
		wg.Add(1)
		urlCh <- url
	}

	close(urlCh)

	wg.Wait()

}
