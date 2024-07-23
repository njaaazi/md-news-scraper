package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"

    "github.com/PuerkitoBio/goquery"
    "github.com/cheggaaa/pb/v3"
)

type Article struct {
    Title         string
    FeaturedImage string
    PublishedDate string
    Content       string
    GalleryImages []string
}

const baseURL = "https://md.rks-gov.net"

func main() {
    // Load the main HTML document
    res, err := http.Get(baseURL + "/page.aspx?id=1,15")
    if err != nil {
        log.Fatal(err)
    }
    defer res.Body.Close()

    if res.StatusCode != 200 {
        log.Fatalf("Error: status code %d when fetching %s\n", res.StatusCode, baseURL+"/page.aspx?id=1,15")
    }

    doc, err := goquery.NewDocumentFromReader(res.Body)
    if err != nil {
        log.Fatal(err)
    }

    var articles []Article

    // Find each article link
    articleSelection := doc.Find("#MainContent_ctl00_pnlLajmet .portfolio-grid")
    totalArticles := articleSelection.Length()

    // Initialize the progress bar
    bar := pb.StartNew(totalArticles)

    // Find each article link and extract the necessary information
    articleSelection.Each(func(i int, s *goquery.Selection) {
        defer bar.Increment()

        var article Article

        // Extract the featured image
        if featuredImage, exists := s.Find(".port-img img").Attr("src"); exists {
            article.FeaturedImage = baseURL + featuredImage
        }

        // Extract the title and URL
        articleURL, exists := s.Find("h3 a").Attr("href")
        if exists {
            article.Title = s.Find("h3 a").Text()

            // Fetch the article page
            fullURL := baseURL + "/" + articleURL // Ensure you use the full URL here
            res, err := http.Get(fullURL)
            if err != nil {
                log.Println("Error fetching article page:", err)
                return
            }
            defer res.Body.Close()

            if res.StatusCode != 200 {
                log.Printf("Error: status code %d when fetching %s\n", res.StatusCode, fullURL)
                return
            }

            articleDoc, err := goquery.NewDocumentFromReader(res.Body)
            if err != nil {
                log.Println("Error parsing article page:", err)
                return
            }

            // Extract the published date from the main page
            article.PublishedDate = strings.TrimSpace(s.Find(".caption .date").Text())

            // Extract the content
            contentStart := articleDoc.Find("div#div_print p.semibold").NextUntil("div.tz-gallery").Text()

            article.Content = strings.TrimSpace(contentStart)

            // Extract gallery images if they exist
            articleDoc.Find("div.tz-gallery a.lightbox img").Each(func(i int, img *goquery.Selection) {
                if imgURL, exists := img.Attr("src"); exists {
                    article.GalleryImages = append(article.GalleryImages, baseURL + imgURL)
                }
            })

            articles = append(articles, article)
        }
    })

    // Finish the progress bar
    bar.Finish()

    // Create CSV file
    file, err := os.Create("articles.csv")
    if err != nil {
        log.Fatalf("failed creating file: %s", err)
    }
    defer file.Close()

    // Write to CSV file
    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Write CSV header
    header := []string{"Title", "FeaturedImage", "PublishedDate", "Content", "GalleryImages"}
    if err := writer.Write(header); err != nil {
        log.Fatalf("failed to write header to csv: %s", err)
    }

    // Write article data to CSV file
    for _, article := range articles {
        record := []string{
            article.Title,
            article.FeaturedImage,
            article.PublishedDate,
            article.Content,
            strings.Join(article.GalleryImages, ","),
        }
        if err := writer.Write(record); err != nil {
            log.Fatalf("failed to write record to csv: %s", err)
        }
    }

    fmt.Println("Data successfully written to articles.csv")
}
