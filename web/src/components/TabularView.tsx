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

import React, { useState } from 'react';
import { Search, ExternalLink, Calendar, Building2, Filter, ArrowUpDown, Sparkles, Radio } from 'lucide-react';

export interface NewsItem {
  id: string;
  run_date: string;
  pub_date: string;
  company: string;
  category: 'Frontier Models' | 'AI Research Papers' | 'AI Business & Infra' | 'OSS & Tooling' | 'Google Cloud';
  title: string;
  summary: string;
  link: string;
  raw_source: string;
}

interface TabularViewProps {
  items: NewsItem[];
  isLoading: boolean;
  onOpenChatWithArticle?: (item: NewsItem) => void;
  onOpenLiveVoiceWithArticle?: (item: NewsItem) => void;
}

export const TabularView: React.FC<TabularViewProps> = ({
  items,
  isLoading,
  onOpenChatWithArticle,
  onOpenLiveVoiceWithArticle,
}) => {
  const [search, setSearch] = useState('');
  const [selectedCompany, setSelectedCompany] = useState<string>('All');
  const [selectedCategory, setSelectedCategory] = useState<string>('All');
  const [sortField, setSortField] = useState<'pub_date' | 'run_date'>('pub_date');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const [selectedItem, setSelectedItem] = useState<NewsItem | null>(null);

  const companies = ['All', 'Google', 'Google Cloud', 'Anthropic', 'OpenAI', 'X AI', 'Meta AI', 'Hugging Face', 'arXiv', 'Business', 'Tooling'];

  // Filter items
  const filteredItems = items.filter((item) => {
    const matchesSearch =
      search === '' ||
      item.title.toLowerCase().includes(search.toLowerCase()) ||
      item.summary.toLowerCase().includes(search.toLowerCase()) ||
      item.company.toLowerCase().includes(search.toLowerCase());

    const matchesCompany = selectedCompany === 'All' || item.company.toLowerCase().includes(selectedCompany.toLowerCase());
    const matchesCategory = selectedCategory === 'All' || item.category === selectedCategory;

    return matchesSearch && matchesCompany && matchesCategory;
  });

  // Sort items
  const sortedItems = [...filteredItems].sort((a, b) => {
    const valA = new Date(a[sortField]).getTime();
    const valB = new Date(b[sortField]).getTime();
    return sortOrder === 'desc' ? valB - valA : valA - valB;
  });

  const toggleSort = (field: 'pub_date' | 'run_date') => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'desc' ? 'asc' : 'desc');
    } else {
      setSortField(field);
      setSortOrder('desc');
    }
  };

  const getCompanyBadgeStyle = (company: string) => {
    const lower = company.toLowerCase();
    if (lower.includes('google cloud') || lower.includes('gcp')) return 'bg-sky-500/10 text-sky-400 border-sky-500/30';
    if (lower.includes('google')) return 'bg-blue-500/10 text-blue-400 border-blue-500/30';
    if (lower.includes('anthropic')) return 'bg-amber-500/10 text-amber-400 border-amber-500/30';
    if (lower.includes('openai')) return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30';
    if (lower.includes('x ai') || lower.includes('xai')) return 'bg-purple-500/10 text-purple-400 border-purple-500/30';
    if (lower.includes('hugging')) return 'bg-yellow-500/10 text-yellow-400 border-yellow-500/30';
    if (lower.includes('arxiv')) return 'bg-fuchsia-500/10 text-fuchsia-400 border-fuchsia-500/30';
    if (lower.includes('business') || lower.includes('deal') || lower.includes('infra')) return 'bg-teal-500/10 text-teal-400 border-teal-500/30';
    return 'bg-slate-700/40 text-slate-300 border-slate-600/40';
  };

  const getCategoryBadgeStyle = (cat: string) => {
    if (cat === 'Google Cloud') return 'bg-sky-500/15 text-sky-300 border-sky-500/40';
    if (cat === 'Frontier Models') return 'bg-blue-500/15 text-blue-300 border-blue-500/40';
    if (cat === 'AI Research Papers') return 'bg-purple-500/15 text-purple-300 border-purple-500/40';
    if (cat === 'AI Business & Infra') return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/40';
    if (cat === 'OSS & Tooling') return 'bg-amber-500/15 text-amber-300 border-amber-500/40';
    return 'bg-slate-800 text-slate-300 border-slate-700';
  };

  return (
    <div className="space-y-6">
      
      {/* Controls Header: Search & Filters */}
      <div className="glass-panel p-5 rounded-2xl space-y-4">
        <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-4">
          
          {/* Search Input */}
          <div className="relative flex-1">
            <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by title, summary, company, or concept..."
              className="w-full bg-slate-900/90 border border-slate-800 rounded-xl pl-10 pr-4 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all"
            />
          </div>

          {/* Category Filter */}
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4 text-slate-400 hidden sm:block" />
            <select
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="bg-slate-900/90 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 cursor-pointer"
            >
              <option value="All">All Intelligence Streams (5 Streams)</option>
              <option value="Frontier Models">🔵 Frontier Models</option>
              <option value="Google Cloud">☁️ Google Cloud Release Notes</option>
              <option value="AI Research Papers">🟣 AI Research Papers</option>
              <option value="AI Business & Infra">🟢 AI Business & Infra</option>
              <option value="OSS & Tooling">🟠 OSS & Tooling</option>
            </select>
          </div>
        </div>

        {/* Company Quick Pills */}
        <div className="flex items-center gap-2 overflow-x-auto pb-1 pt-1 no-scrollbar">
          <span className="text-xs font-semibold text-slate-400 whitespace-nowrap mr-1">Filter Source:</span>
          {companies.map((c) => (
            <button
              key={c}
              onClick={() => setSelectedCompany(c)}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition-all whitespace-nowrap border ${
                selectedCompany === c
                  ? 'bg-indigo-600 text-white border-indigo-500 shadow-md shadow-indigo-600/30'
                  : 'bg-slate-900/80 text-slate-400 border-slate-800 hover:border-slate-700 hover:text-slate-200'
              }`}
            >
              {c}
            </button>
          ))}
        </div>
      </div>

      {/* Main Tabular View Data Table */}
      <div className="glass-panel rounded-2xl overflow-hidden shadow-2xl border border-slate-800/80">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-900/90 border-b border-slate-800 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                <th className="py-4 px-4 cursor-pointer hover:text-slate-200" onClick={() => toggleSort('run_date')}>
                  <div className="flex items-center gap-1.5">
                    <Calendar className="h-3.5 w-3.5" />
                    Run Date
                    <ArrowUpDown className="h-3 w-3 text-slate-500" />
                  </div>
                </th>
                <th className="py-4 px-4 cursor-pointer hover:text-slate-200" onClick={() => toggleSort('pub_date')}>
                  <div className="flex items-center gap-1.5">
                    <Calendar className="h-3.5 w-3.5 text-indigo-400" />
                    Pub Date
                    <ArrowUpDown className="h-3 w-3 text-slate-500" />
                  </div>
                </th>
                <th className="py-4 px-4">
                  <div className="flex items-center gap-1.5">
                    <Building2 className="h-3.5 w-3.5" />
                    Company / Source
                  </div>
                </th>
                <th className="py-4 px-4 max-w-xs">Title & Category</th>
                <th className="py-4 px-4 max-w-md">Summary</th>
                <th className="py-4 px-4 text-right">Links</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 text-sm">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-slate-400">
                    Loading agent data records...
                  </td>
                </tr>
              ) : sortedItems.length === 0 ? (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-slate-400">
                    No model news or papers found matching your criteria.
                  </td>
                </tr>
              ) : (
                sortedItems.map((item) => (
                  <tr
                    key={item.id}
                    onClick={() => setSelectedItem(item)}
                    className="hover:bg-slate-800/40 transition-colors cursor-pointer group"
                  >
                    {/* Run Date */}
                    <td className="py-4 px-4 text-xs font-mono text-slate-400 whitespace-nowrap">
                      {item.run_date}
                    </td>

                    {/* Pub Date */}
                    <td className="py-4 px-4 text-xs font-mono font-semibold text-indigo-300 whitespace-nowrap">
                      {item.pub_date}
                    </td>

                    {/* Company Badge */}
                    <td className="py-4 px-4 whitespace-nowrap">
                      <span className={`inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold border ${getCompanyBadgeStyle(item.company)}`}>
                        {item.company}
                      </span>
                    </td>

                    {/* Title & Category */}
                    <td className="py-4 px-4 max-w-xs">
                      <div className="font-semibold text-slate-100 group-hover:text-indigo-400 transition-colors line-clamp-2">
                        {item.title}
                      </div>
                      <div className="mt-1.5 flex items-center gap-1">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-semibold border ${getCategoryBadgeStyle(item.category)}`}>
                          {item.category}
                        </span>
                      </div>
                    </td>

                    {/* Summary */}
                    <td className="py-4 px-4 max-w-md text-xs text-slate-300 leading-relaxed">
                      <p className="line-clamp-2">{item.summary}</p>
                    </td>

                    {/* Links & Actions */}
                    <td className="py-4 px-4 text-right whitespace-nowrap space-x-2" onClick={(e) => e.stopPropagation()}>
                      {onOpenLiveVoiceWithArticle && (
                        <button
                          type="button"
                          onClick={() => onOpenLiveVoiceWithArticle(item)}
                          className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold bg-purple-600/15 text-purple-300 hover:bg-purple-600 hover:text-white border border-purple-500/30 transition-all shadow-sm"
                          title="Start Gemini Live Voice session about this article"
                        >
                          <Radio className="h-3 w-3 text-purple-400" />
                          <span>Voice</span>
                        </button>
                      )}
                      {onOpenChatWithArticle && (
                        <button
                          type="button"
                          onClick={() => onOpenChatWithArticle(item)}
                          className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold bg-indigo-600/15 text-indigo-300 hover:bg-indigo-600 hover:text-white border border-indigo-500/30 transition-all shadow-sm"
                          title="Ask Gemini Agent about this article"
                        >
                          <Sparkles className="h-3 w-3 text-indigo-400" />
                          <span>Chat</span>
                        </button>
                      )}
                      <a
                        href={item.link}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-semibold bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white border border-slate-700 transition-all shadow-sm"
                      >
                        Visit
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="bg-slate-900/80 px-4 py-3 border-t border-slate-800 text-xs text-slate-400 flex items-center justify-between">
          <span>Showing {sortedItems.length} of {items.length} total agent records</span>
          <span className="font-mono text-[11px] text-slate-500">Database: SQLite WAL &bull; Realtime Filter</span>
        </div>
      </div>

      {/* Item Detail Modal Drawer */}
      {selectedItem && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-2xl w-full p-6 space-y-5 shadow-2xl relative">
            <div className="flex items-start justify-between gap-4">
              <div>
                <span className={`inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold border ${getCompanyBadgeStyle(selectedItem.company)} mb-2`}>
                  {selectedItem.company} &bull; {selectedItem.category}
                </span>
                <h2 className="text-xl font-bold text-slate-100">{selectedItem.title}</h2>
              </div>
              <button
                onClick={() => setSelectedItem(null)}
                className="text-slate-400 hover:text-white text-xl font-semibold px-2"
              >
                ✕
              </button>
            </div>

            <div className="grid grid-cols-2 gap-4 bg-slate-950/60 p-3 rounded-xl border border-slate-800 text-xs">
              <div>
                <span className="text-slate-500">Publication Date: </span>
                <span className="font-mono font-semibold text-slate-200">{selectedItem.pub_date}</span>
              </div>
              <div>
                <span className="text-slate-500">Agent Run Date: </span>
                <span className="font-mono font-semibold text-slate-200">{selectedItem.run_date}</span>
              </div>
              <div>
                <span className="text-slate-500">Source Provider: </span>
                <span className="font-semibold text-indigo-400">{selectedItem.raw_source}</span>
              </div>
              <div>
                <span className="text-slate-500">Record ID: </span>
                <span className="font-mono text-slate-400">{selectedItem.id}</span>
              </div>
            </div>

            <div>
              <h3 className="text-sm font-semibold text-slate-300 mb-2">Executive Summary</h3>
              <p className="text-sm text-slate-300 leading-relaxed bg-slate-950/40 p-4 rounded-xl border border-slate-800/80">
                {selectedItem.summary}
              </p>
            </div>

            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                onClick={() => setSelectedItem(null)}
                className="px-4 py-2 rounded-xl text-sm font-medium text-slate-400 hover:text-slate-200 bg-slate-800/60"
              >
                Close
              </button>

              {onOpenLiveVoiceWithArticle && (
                <button
                  type="button"
                  onClick={() => {
                    const item = selectedItem;
                    setSelectedItem(null);
                    onOpenLiveVoiceWithArticle(item);
                  }}
                  className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-semibold bg-purple-600/20 text-purple-300 hover:bg-purple-600 hover:text-white border border-purple-500/40 transition-all shadow-md"
                >
                  <Radio className="h-4 w-4 text-purple-400 animate-pulse" />
                  <span>Gemini Live Voice</span>
                </button>
              )}

              {onOpenChatWithArticle && (
                <button
                  type="button"
                  onClick={() => {
                    const item = selectedItem;
                    setSelectedItem(null);
                    onOpenChatWithArticle(item);
                  }}
                  className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-semibold bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600 hover:text-white border border-indigo-500/30 transition-all shadow-md"
                >
                  <Sparkles className="h-4 w-4 text-indigo-400" />
                  <span>Chat with AI</span>
                </button>
              )}

              <a
                href={selectedItem.link}
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-1.5 px-5 py-2 rounded-xl text-sm font-semibold bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/30 transition-all"
              >
                Open Source
                <ExternalLink className="h-4 w-4" />
              </a>
            </div>
          </div>
        </div>
      )}

    </div>
  );
};
