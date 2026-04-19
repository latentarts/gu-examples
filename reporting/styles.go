package main

import (
	"github.com/latentart/gu/el"
)

// GlobalStyles returns the global CSS for the reporting application.
func GlobalStyles() el.Node {
	css := `
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			background-color: #020617;
			color: #f1f5f9;
			margin: 0;
			padding: 20px;
		}
		.app {
			max-width: 1200px;
			margin: 0 auto;
		}
		h1 {
			font-weight: 200;
			letter-spacing: -0.025em;
			margin-bottom: 1.5rem;
			color: #f8fafc;
		}
		.uploader {
			margin-bottom: 2rem;
			padding: 2.5rem;
			background: #0f172a;
			border: 2px dashed #334155;
			border-radius: 0.75rem;
			text-align: center;
			transition: all 0.2s ease;
		}
		.uploader:hover {
			border-color: #38bdf8;
			background: #1e293b;
		}
		.uploader p {
			color: #94a3b8;
			margin-bottom: 1rem;
		}
		.table-container {
			background: #0f172a;
			border: 1px solid #1e293b;
			border-radius: 0.5rem;
			overflow: auto;
			max-height: 70vh;
			position: relative;
			box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.2), 0 2px 4px -2px rgba(0, 0, 0, 0.2);
		}
		table {
			width: 100%;
			border-collapse: collapse;
			font-size: 0.875rem;
		}
		thead {
			position: sticky;
			top: 0;
			z-index: 10;
			background: #1e293b;
			box-shadow: 0 1px 0 0 #334155;
		}
		th {
			padding: 1rem;
			text-align: left;
			font-weight: 500;
			color: #f1f5f9;
			cursor: pointer;
			user-select: none;
			white-space: nowrap;
			transition: background 0.15s ease;
		}
		th:hover {
			background: #334155;
		}
		td {
			padding: 1rem;
			border-bottom: 1px solid #1e293b;
			color: #cbd5e1;
			transition: background 0.1s ease;
		}
		tr:hover td {
			background: #1e293b;
			color: #f8fafc;
		}
		.sort-icon {
			margin-left: 0.5rem;
			color: #38bdf8;
		}
		input[type="file"] {
			color: #94a3b8;
			font-size: 0.875rem;
		}
		input[type="file"]::file-selector-button {
			background: #334155;
			color: white;
			border: none;
			padding: 0.5rem 1rem;
			border-radius: 0.375rem;
			cursor: pointer;
			margin-right: 1rem;
			transition: background 0.2s;
		}
		input[type="file"]::file-selector-button:hover {
			background: #475569;
		}
	`
	return el.Tag("style", el.Text(css))
}
