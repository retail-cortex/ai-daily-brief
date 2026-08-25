import React, { useMemo } from 'react';
import { marked } from 'marked';

interface MarkdownRendererProps {
  content: string;
  className?: string;
}

export const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({ content, className = '' }) => {
  const html = useMemo(() => {
    try {
      marked.setOptions({
        gfm: true,
        breaks: true,
      });
      return marked.parse(content || '') as string;
    } catch (err) {
      console.error('Markdown parse error:', err);
      return content;
    }
  }, [content]);

  return (
    <div className={`markdown-body ${className}`}>
      <style>{`
        .markdown-body {
          color: #e2e8f0;
          line-height: 1.65;
          font-size: 0.8125rem;
        }
        .markdown-body h1,
        .markdown-body h2,
        .markdown-body h3,
        .markdown-body h4 {
          color: #f8fafc;
          font-weight: 700;
          margin-top: 1.1em;
          margin-bottom: 0.45em;
          line-height: 1.35;
        }
        .markdown-body h1 {
          font-size: 1.15rem;
          border-bottom: 1px solid #334155;
          padding-bottom: 0.3em;
          color: #a5b4fc;
        }
        .markdown-body h2 {
          font-size: 1.05rem;
          border-bottom: 1px solid #1e293b;
          padding-bottom: 0.25em;
          color: #93c5fd;
        }
        .markdown-body h3 {
          font-size: 0.95rem;
          color: #c4b5fd;
        }
        .markdown-body h4 {
          font-size: 0.875rem;
          color: #67e8f9;
        }
        .markdown-body p {
          margin-top: 0.4em;
          margin-bottom: 0.6em;
        }
        .markdown-body strong {
          color: #ffffff;
          font-weight: 700;
        }
        .markdown-body em {
          color: #cbd5e1;
          font-style: italic;
        }
        .markdown-body ul {
          list-style-type: disc !important;
          margin-top: 0.4em;
          margin-bottom: 0.6em;
          padding-left: 1.4em !important;
        }
        .markdown-body ol {
          list-style-type: decimal !important;
          margin-top: 0.4em;
          margin-bottom: 0.6em;
          padding-left: 1.4em !important;
        }
        .markdown-body ul ul,
        .markdown-body ol ul {
          list-style-type: circle !important;
          margin-top: 0.2em;
          margin-bottom: 0.2em;
          padding-left: 1.2em !important;
        }
        .markdown-body li {
          margin-top: 0.25em;
          margin-bottom: 0.25em;
          color: #e2e8f0;
        }
        .markdown-body hr {
          border: 0;
          height: 1px;
          background-color: #334155;
          margin: 1em 0;
        }
        .markdown-body blockquote {
          border-left: 3px solid #6366f1;
          background-color: rgba(30, 27, 75, 0.4);
          padding: 0.5em 1em;
          margin: 0.6em 0;
          border-radius: 0 8px 8px 0;
          color: #c7d2fe;
        }
        .markdown-body code {
          font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
          background-color: #0f172a;
          border: 1px solid #334155;
          color: #a5b4fc;
          padding: 0.15em 0.4em;
          border-radius: 4px;
          font-size: 0.85em;
        }
        .markdown-body pre {
          background-color: #020617;
          border: 1px solid #334155;
          border-radius: 10px;
          padding: 0.85em 1em;
          overflow-x: auto;
          margin: 0.7em 0;
        }
        .markdown-body pre code {
          background-color: transparent;
          border: none;
          padding: 0;
          color: #e2e8f0;
          font-size: 0.825em;
          line-height: 1.5;
        }
        .markdown-body table {
          width: 100%;
          border-collapse: collapse;
          margin: 0.8em 0;
          font-size: 0.85em;
        }
        .markdown-body th,
        .markdown-body td {
          border: 1px solid #334155;
          padding: 0.45em 0.75em;
          text-align: left;
        }
        .markdown-body th {
          background-color: #1e293b;
          color: #f1f5f9;
          font-weight: 600;
        }
        .markdown-body tr:nth-child(even) {
          background-color: rgba(15, 23, 42, 0.5);
        }
        .markdown-body a {
          color: #38bdf8;
          text-decoration: underline;
          text-underline-offset: 2px;
        }
        .markdown-body a:hover {
          color: #7dd3fc;
        }
      `}</style>
      <div dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  );
};
