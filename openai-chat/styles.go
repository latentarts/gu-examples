//go:build js && wasm

package main

import "github.com/latentart/gu/el"

func GlobalStyles() el.Node {
	css := `
        /* Typography & Scrollbars */
        .prose-chat pre {
            background: #09090b !important;
            border: 1px solid #27272a;
            border-radius: 0.75rem;
            padding: 1rem;
            margin: 0.75rem 0;
            overflow-x: auto;
        }
        .prose-chat code {
            font-family: 'JetBrains Mono', ui-monospace, monospace;
            font-size: 0.85em;
            color: #67e8f9;
        }
        .prose-chat p { margin-bottom: 0.75rem; }
        .prose-chat p:last-child { margin-bottom: 0; }
        .prose-chat ul, .prose-chat ol { margin-left: 1.25rem; margin-bottom: 0.75rem; }

        ::-webkit-scrollbar { width: 6px; height: 6px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: #27272a; border-radius: 10px; }
        ::-webkit-scrollbar-thumb:hover { background: #3f3f46; }

        /* Animations */
        @keyframes thinking-bounce {
            0%, 100% { transform: translateY(0); opacity: 0.4; }
            50% { transform: translateY(-3px); opacity: 1; }
        }
        .thinking-dot {
            animation: thinking-bounce 1.4s infinite ease-in-out;
        }
        .thinking-dot:nth-child(2) { animation-delay: 0.2s; }
        .thinking-dot:nth-child(3) { animation-delay: 0.4s; }

        @keyframes shimmer {
            0% { opacity: 1; }
            50% { opacity: 0.6; }
            100% { opacity: 1; }
        }
        .thinking-shimmer {
            animation: shimmer 2s infinite ease-in-out;
        }

        .group[open] .thinking-chevron {
            transform: rotate(180deg);
        }
        .thinking-chevron {
            transition: transform 0.2s ease;
        }
	`
	return el.Tag("style", el.Text(css))
}
