package main

import (
	"github.com/latentart/gu/el"
)

// GlobalStyles returns the global CSS for the counter application.
func GlobalStyles() el.Node {
	css := `
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: var(--gu-font-family, system-ui, sans-serif);
            background: var(--gu-color-background, #ffffff);
            color: var(--gu-color-text-primary, #0f172a);
            line-height: var(--gu-line-height, 1.5);
            transition: background-color 0.2s, color 0.2s;
        }
        .app {
            max-width: 600px;
            margin: 0 auto;
            padding: 2rem;
            text-align: center;
        }
        h1 {
            margin-bottom: 1.5rem;
            color: var(--gu-color-primary, #2563eb);
        }
        p {
            font-size: 1.25rem;
            margin-bottom: 0.75rem;
        }
        .buttons {
            display: flex;
            gap: 0.5rem;
            justify-content: center;
            margin: 1.5rem 0;
        }
        button {
            padding: 0.5rem 1.25rem;
            border: 1px solid var(--gu-color-border, #e2e8f0);
            border-radius: 0.375rem;
            background: var(--gu-color-primary, #2563eb);
            color: white;
            font-size: 1rem;
            cursor: pointer;
            transition: opacity 0.15s;
        }
        button:hover { opacity: 0.85; }
        .theme-toggle {
            margin-top: 1.5rem;
            background: var(--gu-color-secondary, #7c3aed);
        }
	`
	return el.Tag("style", el.Text(css))
}
