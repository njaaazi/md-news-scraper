package main

import (
    "encoding/csv"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "sort"
    "strings"
    "time"

    "github.com/PuerkitoBio/goquery"
    "github.com/cheggaaa/pb/v3"
)

type Article struct {
    Title         string
    FeaturedImage string
    PublishedDate string
    Content       string
    GalleryImages []string
    DateTime      time.Time
}

const baseURL = "https://md.rks-gov.net"

func main() {
    // Define command line argument
    maxArticlesPtr := flag.Int("num", -1, "The number of articles to crawl (default: all articles)")
    flag.Parse()

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
    articleCount := 0

    // Find each article link
    articleSelection := doc.Find("#MainContent_ctl00_pnlLajmet .portfolio-grid")
    totalArticles := articleSelection.Length()

    maxArticles := *maxArticlesPtr
    if maxArticles == -1 || maxArticles > totalArticles {
        maxArticles = totalArticles
    }

    // Initialize the progress bar
    bar := pb.StartNew(maxArticles)

    // Find each article link and extract the necessary information
    articleSelection.Each(func(i int, s *goquery.Selection) {
        if articleCount >= maxArticles {
            return
        }
        defer bar.Increment()

        var article Article

        // Extract the featured image
        if featuredImage, exists := s.Find(".port-img img").Attr("src"); exists {
            article.FeaturedImage = baseURL + featuredImage
        }

        // Extract the URL
        articleURL, exists := s.Find("h3 a").Attr("href")
        if exists {
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

            // Extract the title from inside the article
            article.Title = articleDoc.Find("h3").Text()

            // Extract the published date and time from the main page
            publishedDate := strings.TrimSpace(s.Find(".caption .date").Text())
            // Extract only the date part
            parts := strings.Split(publishedDate, " ")
            article.PublishedDate = parts[len(parts)-1]

            // Parse the date and time
            layout := "02/01/2006"
            article.DateTime, err = time.Parse(layout, article.PublishedDate)
            if err != nil {
                log.Println("Error parsing date:", err)
                return
            }

            // Extract the content with HTML tags
            contentSelection := articleDoc.Find("div#div_print p.semibold").NextUntil("div.tz-gallery")
            var contentHTML strings.Builder
            contentSelection.Each(func(i int, p *goquery.Selection) {
                html, err := goquery.OuterHtml(p)
                if err != nil {
                    log.Println("Error extracting content HTML:", err)
                    return
                }
                // Check if the paragraph is empty
                if strings.TrimSpace(p.Text()) != "" {
                    contentHTML.WriteString(html)
                }
            })
            article.Content = contentHTML.String()

            // Extract gallery images if they exist
            articleDoc.Find("div.tz-gallery a.lightbox img").Each(func(i int, img *goquery.Selection) {
                if imgURL, exists := img.Attr("src"); exists {
                    article.GalleryImages = append(article.GalleryImages, baseURL+imgURL)
                }
            })

            articles = append(articles, article)
            articleCount++
        }
    })

    // Finish the progress bar
    bar.Finish()

    // Sort articles by DateTime in descending order
    sort.Slice(articles, func(i, j int) bool {
        return articles[i].DateTime.After(articles[j].DateTime)
    })

    // Adjust the time for each subsequent article to ensure they are in descending order
    currentTime := time.Now()
    for i := range articles {
        articles[i].PublishedDate = articles[i].DateTime.Format("02/01/2006") + " " + currentTime.Format("15:04")
        currentTime = currentTime.Add(-time.Minute)
    }

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