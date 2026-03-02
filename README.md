# Matthew Fritsch Portfolio

Static portfolio website for GitHub Pages.

## Local Development

Simply open `index.html` in your browser, or use a local server:

```bash
python3 -m http.server 8000
```

Then visit `http://localhost:8000`

## Deploy to GitHub Pages

1. Create a new repository on GitHub (e.g., `matthewfritsch.github.io` or any repo name)
2. Push this code to the repository:
   ```bash
   git init
   git add .
   git commit -m "Initial portfolio"
   git branch -M main
   git remote add origin https://github.com/yourusername/yourrepo.git
   git push -u origin main
   ```
3. Go to repository Settings → Pages
4. Under "Source", select "Deploy from a branch"
5. Select the `main` branch and `/ (root)` folder
6. Click Save

Your site will be live at `https://yourusername.github.io/yourrepo/` (or `https://yourusername.github.io` if using username.github.io repo)

## Features

- Fully static (no build process needed)
- Dark/light theme toggle with localStorage persistence
- Live local time display (Pacific timezone)
- Responsive design
- Gradient background matching original design
- Markdown blog with syntax highlighting

## Blog

### Writing a Post

Create a new `.md` file in `content/posts/` with YAML frontmatter:

```markdown
---
title: "My Post Title"
date: 2025-03-15
description: "A short description for the listing page."
---

Your markdown content here...
```

### Building Locally

```bash
go build -o generate ./cmd/generate && ./generate
```

This generates the static site in `dist/`.

### Previewing

```bash
python3 -m http.server 8000 -d dist
```

Then visit `http://localhost:8000`

### Deployment

Push to `main` and GitHub Actions will automatically build and deploy.

**Note:** GitHub Pages source must be set to "GitHub Actions" in repo Settings → Pages.
