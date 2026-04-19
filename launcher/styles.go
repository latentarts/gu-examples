package main

import (
	"github.com/latentart/gu/el"
)

// GlobalStyles returns the global CSS for the launcher application.
func GlobalStyles() el.Node {
	css := `
	* { margin: 0; padding: 0; box-sizing: border-box; }
	body {
		font-family: var(--gu-font-family, system-ui, -apple-system, sans-serif);
		background: #0a0a0f;
		color: #e2e8f0;
		line-height: 1.5;
		overflow: hidden;
		height: 100vh;
	}

	/* Layout */
	.launcher {
		display: flex;
		height: 100vh;
		overflow: hidden;
	}
	.launcher__sidebar {
		width: 340px;
		flex-shrink: 0;
		border-right: 1px solid #1e293b;
		overflow-y: auto;
		background: #0f1117;
	}
	.launcher__main {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		position: relative;
	}

	/* Cards */
	.example-card {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		cursor: pointer;
		border-bottom: 1px solid #1e293b;
		transition: background-color 0.15s;
	}
	.example-card:hover {
		background-color: #1a1d2e;
	}
	.example-card--selected {
		background-color: #1a1d2e;
		border-left: 3px solid #3b82f6;
	}
	.example-card__thumb {
		width: 64px;
		height: 48px;
		border-radius: 6px;
		object-fit: cover;
		flex-shrink: 0;
		background: #1e293b;
	}
	.example-card__info {
		min-width: 0;
	}
	.example-card__name {
		font-size: 14px;
		font-weight: 600;
		color: #f1f5f9;
		margin-bottom: 2px;
	}
	.example-card__desc {
		font-size: 12px;
		color: #94a3b8;
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	/* Sidebar */
	.sidebar {
		display: flex;
		flex-direction: column;
		height: 100%;
	}
	.sidebar__spacer {
		flex: 1;
	}
	.sidebar__footer {
		padding: 12px 16px;
		border-top: 1px solid #1e293b;
	}
	.sidebar__link {
		color: #64748b;
		text-decoration: none;
		font-size: 12px;
	}
	.sidebar__link:hover {
		color: #94a3b8;
	}

	/* Viewer */
	.viewer {
		display: flex;
		flex-direction: column;
		height: 100%;
	}
	.viewer__header {
		padding: 12px 20px;
		background: #0f1117;
		border-bottom: 1px solid #1e293b;
		flex-shrink: 0;
	}
	.viewer__title {
		font-size: 16px;
		font-weight: 600;
		color: #f1f5f9;
	}
	.viewer__desc {
		font-size: 13px;
		color: #94a3b8;
		margin-top: 2px;
	}
	.viewer__iframe {
		flex: 1;
		border: none;
		width: 100%;
		height: 100%;
		background: #ffffff;
	}

	/* Welcome */
	.welcome {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
	}
	.welcome__content {
		text-align: center;
		padding: 40px;
	}
	.welcome__title {
		font-size: 28px;
		font-weight: 700;
		color: #f1f5f9;
		margin-bottom: 8px;
	}
	.welcome__subtitle {
		font-size: 16px;
		color: #94a3b8;
		margin-bottom: 24px;
	}
	.welcome__stats {
		color: #64748b;
		font-size: 14px;
	}
	.welcome__stat {
		display: inline-block;
		padding: 6px 16px;
		background: #1a1d2e;
		border-radius: 20px;
		border: 1px solid #1e293b;
	}

	/* Mobile toggle */
	.mobile-toggle {
		display: none;
		position: absolute;
		top: 12px;
		left: 12px;
		z-index: 10;
		background: #1e293b;
		color: #e2e8f0;
		border: 1px solid #334155;
		border-radius: 6px;
		padding: 6px 10px;
		font-size: 18px;
		cursor: pointer;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.launcher {
			flex-direction: column;
		}
		.launcher__sidebar {
			width: 100%;
			max-height: 50vh;
			border-right: none;
			border-bottom: 1px solid #1e293b;
		}
		.mobile-toggle {
			display: block;
		}
	}
	`

	return el.Tag("style", el.Text(css))
}