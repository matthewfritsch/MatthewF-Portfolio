package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"gopkg.in/yaml.v3"
)

// FrontMatter holds parsed YAML frontmatter from a markdown post.
type FrontMatter struct {
	Title       string `yaml:"title"`
	Date        string `yaml:"date"`
	Description string `yaml:"description"`
}

// Post represents a fully parsed blog post ready for rendering.
type Post struct {
	Title         string
	Date          time.Time
	DateFormatted string
	Description   string
	Content       template.HTML
	Slug          string
	URL           string
}

// PostPageData is the template data for a single post page.
type PostPageData struct {
	Title         string
	DateFormatted string
	Description   string
	Content       template.HTML
	RelativeRoot  string
	URL           string
}

// ListingPageData is the template data for the blog listing page.
type ListingPageData struct {
	Posts        []Post
	RelativeRoot string
}

func main() {
	const (
		contentDir  = "content/posts"
		templateDir = "templates"
		distDir     = "dist"
	)

	// Parse templates
	tmplFiles := []string{
		filepath.Join(templateDir, "post.html"),
		filepath.Join(templateDir, "listing.html"),
	}
	for _, tf := range tmplFiles {
		if _, err := os.Stat(tf); os.IsNotExist(err) {
			log.Fatalf("missing required template: %s", tf)
		}
	}
	templates := template.Must(template.ParseFiles(tmplFiles...))

	// Configure goldmark with GFM + syntax highlighting (class-based)
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
	)

	// Read markdown posts
	posts, err := readPosts(contentDir, md)
	if err != nil {
		log.Fatalf("error reading posts: %v", err)
	}

	// Sort posts by date descending
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	// Clean and create dist directory
	if err := os.RemoveAll(distDir); err != nil {
		log.Fatalf("error cleaning dist: %v", err)
	}
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		log.Fatalf("error creating dist: %v", err)
	}

	// Check for duplicate URL paths
	seen := make(map[string]string)
	for _, p := range posts {
		if prev, ok := seen[p.URL]; ok {
			log.Fatalf("duplicate post URL %s: %s and %s", p.URL, prev, p.Slug+".md")
		}
		seen[p.URL] = p.Slug + ".md"
	}

	// Generate blog listing page
	listingDir := filepath.Join(distDir, "blog")
	if err := os.MkdirAll(listingDir, 0o755); err != nil {
		log.Fatalf("error creating blog dir: %v", err)
	}
	listingData := ListingPageData{
		Posts:        posts,
		RelativeRoot: "../",
	}
	if err := writeTemplate(templates, "listing.html", filepath.Join(listingDir, "index.html"), listingData); err != nil {
		log.Fatalf("error writing listing page: %v", err)
	}

	// Generate individual post pages
	for _, p := range posts {
		postDir := filepath.Join(distDir, p.URL)
		if err := os.MkdirAll(postDir, 0o755); err != nil {
			log.Fatalf("error creating post dir %s: %v", postDir, err)
		}
		// Calculate relative root from post depth: blog/YYYY/MM/slug/ = 4 levels
		relRoot := "../../../../"
		pageData := PostPageData{
			Title:         p.Title,
			DateFormatted: p.DateFormatted,
			Description:   p.Description,
			Content:       p.Content,
			RelativeRoot:  relRoot,
			URL:           p.URL,
		}
		if err := writeTemplate(templates, "post.html", filepath.Join(postDir, "index.html"), pageData); err != nil {
			log.Fatalf("error writing post %s: %v", p.Slug, err)
		}
	}

	// Copy static files to dist
	staticFiles := []string{"index.html", "style.css", "script.js", "blog.css"}
	for _, sf := range staticFiles {
		if _, err := os.Stat(sf); os.IsNotExist(err) {
			continue // Skip missing optional files (e.g., blog.css may not exist yet)
		}
		if err := copyFile(sf, filepath.Join(distDir, sf)); err != nil {
			log.Fatalf("error copying %s: %v", sf, err)
		}
	}

	// Copy assets directory recursively
	if _, err := os.Stat("assets"); err == nil {
		if err := copyDir("assets", filepath.Join(distDir, "assets")); err != nil {
			log.Fatalf("error copying assets: %v", err)
		}
	}

	// Generate Chroma CSS for syntax highlighting
	if err := writeChromaCSS(distDir); err != nil {
		log.Fatalf("error generating syntax CSS: %v", err)
	}

	// Write .nojekyll file
	if err := os.WriteFile(filepath.Join(distDir, ".nojekyll"), []byte{}, 0o644); err != nil {
		log.Fatalf("error writing .nojekyll: %v", err)
	}

	fmt.Printf("Site generated: %d posts written to %s/\n", len(posts), distDir)
}

// readPosts reads all .md files from dir, parses frontmatter and renders markdown.
func readPosts(dir string, md goldmark.Markdown) ([]Post, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // Missing dir treated as empty
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	var posts []Post
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		fm, body, err := parseFrontmatter(data, entry.Name())
		if err != nil {
			log.Fatalf("error parsing frontmatter in %s: %v", entry.Name(), err)
		}

		date, err := time.Parse("2006-01-02", fm.Date)
		if err != nil {
			log.Fatalf("error parsing date in %s: %v (expected YYYY-MM-DD)", entry.Name(), err)
		}

		var htmlBuf bytes.Buffer
		if err := md.Convert(body, &htmlBuf); err != nil {
			return nil, fmt.Errorf("rendering markdown in %s: %w", entry.Name(), err)
		}

		slug := strings.TrimSuffix(entry.Name(), ".md")
		url := fmt.Sprintf("blog/%04d/%02d/%s/", date.Year(), int(date.Month()), slug)

		posts = append(posts, Post{
			Title:         fm.Title,
			Date:          date,
			DateFormatted: date.Format("January 2, 2006"),
			Description:   fm.Description,
			Content:       template.HTML(htmlBuf.String()),
			Slug:          slug,
			URL:           url,
		})
	}
	return posts, nil
}

// parseFrontmatter splits markdown content on --- delimiters and unmarshals YAML.
func parseFrontmatter(data []byte, filename string) (FrontMatter, []byte, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return FrontMatter{}, nil, fmt.Errorf("missing opening --- delimiter")
	}

	// Find closing delimiter
	rest := content[4:] // skip opening "---\n"
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		// Try ending with --- at EOF (no trailing newline after closing ---)
		idx = strings.Index(rest, "\n---")
		if idx < 0 || (idx+4 < len(rest) && rest[idx+4] != '\n') {
			return FrontMatter{}, nil, fmt.Errorf("missing closing --- delimiter")
		}
	}

	yamlStr := rest[:idx]
	body := []byte(rest[idx+4:]) // skip "\n---"
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:] // skip extra newline after closing ---
	}

	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
		return FrontMatter{}, nil, err
	}

	return fm, body, nil
}

// writeTemplate executes a named template and writes the result to a file.
func writeTemplate(templates *template.Template, name, outPath string, data interface{}) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return templates.ExecuteTemplate(f, name, data)
}

// writeChromaCSS generates Chroma CSS files for dark and light themes.
func writeChromaCSS(distDir string) error {
	formatter := chromahtml.New(chromahtml.WithClasses(true))

	// Dark theme (monokai)
	darkStyle := styles.Get("monokai")
	darkFile, err := os.Create(filepath.Join(distDir, "syntax.css"))
	if err != nil {
		return fmt.Errorf("creating syntax.css: %w", err)
	}
	defer darkFile.Close()
	if err := formatter.WriteCSS(darkFile, darkStyle); err != nil {
		return fmt.Errorf("writing dark syntax CSS: %w", err)
	}

	// Light theme (github)
	lightStyle := styles.Get("github")
	lightFile, err := os.Create(filepath.Join(distDir, "syntax-light.css"))
	if err != nil {
		return fmt.Errorf("creating syntax-light.css: %w", err)
	}
	defer lightFile.Close()
	if err := formatter.WriteCSS(lightFile, lightStyle); err != nil {
		return fmt.Errorf("writing light syntax CSS: %w", err)
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		return copyFile(path, dstPath)
	})
}
