//go:build js && wasm

package main

import "github.com/latentart/gu/el"

// GlobalStyles returns the global CSS for the DuckDB application.
func GlobalStyles() el.Node {
	css := `
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: var(--gu-font-family, system-ui, sans-serif);
            background: var(--gu-color-background, #0f172a);
            color: var(--gu-color-text-primary, #e2e8f0);
            line-height: var(--gu-line-height, 1.5);
        }
        .app {
            max-width: 960px;
            margin: 0 auto;
            padding: 2rem;
        }
        h1 {
            color: #f59e0b;
            margin-bottom: 0.5rem;
        }
        .subtitle {
            color: #94a3b8;
            margin-bottom: 1.5rem;
        }
        textarea {
            width: 100%;
            min-height: 100px;
            padding: 0.75rem;
            font-family: var(
                --gu-font-family-mono,
                ui-monospace,
                monospace
            );
            font-size: 0.875rem;
            border: 1px solid #334155;
            border-radius: 0.5rem;
            background: #1e293b;
            color: #e2e8f0;
            resize: vertical;
        }
        textarea:focus {
            outline: none;
            border-color: #f59e0b;
        }
        .toolbar {
            display: flex;
            gap: 0.5rem;
            margin: 0.75rem 0;
            align-items: center;
        }
        button {
            padding: 0.5rem 1.25rem;
            border: none;
            border-radius: 0.375rem;
            background: #f59e0b;
            color: #0f172a;
            font-weight: 600;
            font-size: 0.875rem;
            cursor: pointer;
            transition: opacity 0.15s;
        }
        button:hover {
            opacity: 0.85;
        }
        button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .status {
            font-size: 0.8rem;
            color: #94a3b8;
        }
        .error {
            padding: 0.75rem;
            margin: 0.75rem 0;
            border-radius: 0.375rem;
            background: #7f1d1d;
            color: #fca5a5;
            font-size: 0.875rem;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 1rem;
            font-size: 0.875rem;
        }
        th {
            text-align: left;
            padding: 0.5rem 0.75rem;
            background: #1e293b;
            border-bottom: 2px solid #f59e0b;
            color: #f59e0b;
            font-weight: 600;
            position: sticky;
            top: 0;
        }
        td {
            padding: 0.5rem 0.75rem;
            border-bottom: 1px solid #1e293b;
        }
        tr:hover td {
            background: #1e293b44;
        }
        .results {
            max-height: 500px;
            overflow: auto;
            border-radius: 0.5rem;
        }
        .loading-spinner {
            display: inline-block;
            animation: spin 1s linear infinite;
        }
        @keyframes spin {
            to {
                transform: rotate(360deg);
            }
        }
        `
	return el.Tag("style", el.Text(css))
}

