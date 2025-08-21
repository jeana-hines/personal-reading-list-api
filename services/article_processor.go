package services

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/jeana-hines/personal-reading-list-api/models"
	"google.golang.org/genai"
)

// ArticleProcessor handles the processing of articles, including summarization and tagging.
func ProcessNewArticle(article *models.Article) {
	log.Printf("Starting background processing for article ID: %s", article.ID)

	// 1. Fetch the content
	fullContent, err := http.Get(article.URL)
	if err != nil {
		log.Printf("Failed to fetch content for article %s: %v", article.ID, err)
		// You might want to update the article status to "failed" here
		article.Status = "failed"
		err = article.Save()
		if err != nil {
			log.Printf("Failed to update article status to 'failed' for article %s: %v", article.ID, err)
		}
		// Exit the function early if fetching content fails
		return
	}
	defer fullContent.Body.Close()
	if fullContent.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch content for article %s: HTTP %d", article.ID, fullContent.StatusCode)
		return
	}
	body, err := io.ReadAll(fullContent.Body)
	if err != nil {
		log.Printf("Failed to read content for article %s: %v", article.ID, err)
		return

	}
	// Parse the content with goquery

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		log.Printf("Failed to parse content for article %s: %v", article.ID, err)
		return
	}
	// Extract the title and body text
	title := doc.Find("title").Text()
	bodyText := doc.Find("body").Text()
	if title == "" {
		log.Printf("No title found for article %s, using URL as title", article.ID)
		title = article.URL
	}
	article.Title = title
	article.URL = fullContent.Request.URL.String() // Normalize URL

	// 2. Summarize the content (using a hypothetical API call)
	// summary, err := callSummarizationAPI(fullContent)
	// if err != nil { ... }
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}
	config := &genai.ClientConfig{
		APIKey: apiKey,
	}
	client, err := genai.NewClient(ctx, config)
	if err != nil {
		log.Printf("Failed to create GenAI client: %v", err)
		return
	}
	// Generate summary
	summaryResponse, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text("Summarize the following article: "+bodyText), nil)
	if err != nil {
		log.Printf("Failed to summarize article %s: %v", article.ID, err)
		return
	}
	if summaryResponse == nil {
		log.Printf("No summary generated for article %s", article.ID)
		return
	}
	summaryText := summaryResponse.Text()

	// Generate tags
	tagsResponse, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text("Generate a comma-separated list of tags for the following article: "+bodyText), nil)
	if err != nil {
		log.Printf("Failed to get tags for %s: %v", article.ID, err)
		return
	}
	if tagsResponse == nil {
		log.Printf("No tags generated for article %s", article.ID)
		return
	}
	tagsText := tagsResponse.Text()

	// 3. Update the article in the database
	article.Summary = string(summaryText)               // Convert the genai.Text to a string
	article.Tags = strings.Split(string(tagsText), ",") // Split the comma-separated string into a slice of strings
	article.Status = "unread"                           // Or "processed", "read", etc.

	err = article.Save()
	if err != nil {
		log.Printf("Failed to save processed article %s: %v", article.ID, err)
		return
	}

	log.Printf("Successfully processed and updated article ID: %s", article.ID)
}
