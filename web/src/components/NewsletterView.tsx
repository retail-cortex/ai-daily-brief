/**
 * Copyright 2026 Retail Cortex
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState, useEffect } from 'react';
import { Mail, Monitor, Smartphone, RefreshCw, Copy, Download, Code, Sparkles } from 'lucide-react';

const API_BASE = typeof window !== 'undefined' && window.location.port === '5173'
  ? 'http://localhost:3001/api'
  : '/api';

export const NewsletterView: React.FC = () => {
  const [htmlContent, setHtmlContent] = useState<string>('');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isGeneratingTLDR, setIsGeneratingTLDR] = useState<boolean>(false);
  const [previewMode, setPreviewMode] = useState<'desktop' | 'mobile'>('desktop');
  const [showCode, setShowCode] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);

  const fetchPreview = async () => {
    setIsLoading(true);
    try {
      const res = await fetch(`${API_BASE}/newsletter/preview`, { method: 'POST' });
      const data = await res.json();
      if (data.success) {
        setHtmlContent(data.html);
      }
    } catch (err) {
      console.error('Error fetching preview:', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchPreview();
  }, []);

  const handleGenerateTLDR = async () => {
    setIsGeneratingTLDR(true);
    try {
      await fetch(`${API_BASE}/agent/tldr`, { method: 'POST' });
      await fetchPreview();
    } catch (err) {
      console.error('Error generating TLDR:', err);
    } finally {
      setIsGeneratingTLDR(false);
    }
  };

  const copyCodeToClipboard = () => {
    navigator.clipboard.writeText(htmlContent);
    setCopied(true);
    setTimeout(() => setCopied(false), 2500);
  };

  const downloadHtmlFile = () => {
    const dateStr = new Date().toISOString().split('T')[0];
    const blob = new Blob([htmlContent], { type: 'text/html;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `ai_cloud_intelligence_digest_${dateStr}.html`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      
      {/* Control Banner */}
      <div className="glass-panel p-5 rounded-2xl flex flex-col md:flex-row items-stretch md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Mail className="h-5 w-5 text-indigo-400" />
            <h2 className="text-lg font-bold text-slate-100">Daily Executive Digest</h2>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Formatted briefing across Frontier Models, Google Cloud, AI Papers, Business & Tooling ready to export.
          </p>
        </div>

        <div className="flex items-center gap-2.5 flex-wrap">
          
          {/* View Mode Toggle */}
          <div className="flex items-center bg-slate-900 border border-slate-800 rounded-xl p-1">
            <button
              onClick={() => setPreviewMode('desktop')}
              className={`p-2 rounded-lg text-xs font-medium flex items-center gap-1 transition-all ${
                previewMode === 'desktop' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
              }`}
              title="Desktop View"
            >
              <Monitor className="h-4 w-4" />
            </button>
            <button
              onClick={() => setPreviewMode('mobile')}
              className={`p-2 rounded-lg text-xs font-medium flex items-center gap-1 transition-all ${
                previewMode === 'mobile' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
              }`}
              title="Mobile View"
            >
              <Smartphone className="h-4 w-4" />
            </button>
          </div>

          <button
            onClick={() => setShowCode(!showCode)}
            className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200 transition-all text-xs font-medium flex items-center gap-1.5"
          >
            <Code className="h-4 w-4" />
            {showCode ? 'Preview' : 'HTML Code'}
          </button>

          <button
            onClick={fetchPreview}
            className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200 transition-all text-xs font-medium flex items-center gap-1.5"
            title="Refresh Preview"
          >
            <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
          </button>

          {/* Generate Gemini TLDR Button */}
          <button
            onClick={handleGenerateTLDR}
            disabled={isGeneratingTLDR}
            className="flex items-center gap-1.5 px-3.5 py-2.5 rounded-xl font-semibold text-xs bg-indigo-600/15 text-indigo-300 hover:bg-indigo-600/25 border border-indigo-500/30 transition-all active:scale-95 disabled:opacity-50"
            title="Generate or update LLM TLDR analysis with Gemini 3.7"
          >
            <Sparkles className={`h-3.5 w-3.5 text-indigo-400 ${isGeneratingTLDR ? 'animate-spin' : ''}`} />
            <span>{isGeneratingTLDR ? 'Analyzing with Gemini...' : 'Synthesize TL;DR'}</span>
          </button>

          {/* Copy HTML Button */}
          <button
            onClick={copyCodeToClipboard}
            className="flex items-center gap-1.5 px-4 py-2.5 rounded-xl font-semibold text-xs bg-slate-900 border border-slate-700 hover:border-slate-600 text-slate-200 shadow-md transition-all active:scale-95"
          >
            <Copy className="h-3.5 w-3.5 text-indigo-400" />
            {copied ? 'Copied HTML!' : 'Copy HTML'}
          </button>

          {/* Download HTML Button */}
          <button
            onClick={downloadHtmlFile}
            className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-semibold text-xs bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white shadow-lg shadow-indigo-600/25 transition-all active:scale-95"
          >
            <Download className="h-4 w-4" />
            Download .html
          </button>
        </div>
      </div>

      {/* Main Preview Frame / HTML Code View */}
      <div className="flex justify-center transition-all">
        <div
          className={`w-full transition-all duration-300 ${
            previewMode === 'mobile' ? 'max-w-md' : 'max-w-4xl'
          }`}
        >
          <div className="glass-panel rounded-2xl overflow-hidden shadow-2xl border border-slate-800">
            <div className="bg-slate-900 px-4 py-3 border-b border-slate-800 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="h-3 w-3 rounded-full bg-red-500/80 inline-block"></span>
                <span className="h-3 w-3 rounded-full bg-yellow-500/80 inline-block"></span>
                <span className="h-3 w-3 rounded-full bg-green-500/80 inline-block"></span>
                <span className="text-xs font-mono text-slate-400 ml-2">
                  HTML Email Renderer - {previewMode.toUpperCase()} ({previewMode === 'mobile' ? '380px' : 'Full Width'})
                </span>
              </div>
              <button
                onClick={copyCodeToClipboard}
                className="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1 font-medium"
              >
                <Copy className="h-3.5 w-3.5" />
                Copy HTML
              </button>
            </div>

            {showCode ? (
              <pre className="p-4 text-xs font-mono bg-slate-950 text-slate-300 overflow-x-auto max-h-[650px] leading-relaxed">
                {htmlContent}
              </pre>
            ) : (
              <div className="bg-slate-950 p-2 min-h-[650px]">
                <iframe
                  title="Daily Newsletter Live Preview"
                  srcDoc={htmlContent}
                  className="w-full h-[650px] rounded-xl border-0 bg-slate-900"
                />
              </div>
            )}
          </div>
        </div>
      </div>

    </div>
  );
};
