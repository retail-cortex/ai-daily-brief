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

import React from 'react';
import { Bot, Play, Mail, Table, RefreshCw, Sparkles, Settings as SettingsIcon, Radio } from 'lucide-react';

interface HeaderProps {
  activeTab: 'tabular' | 'newsletter';
  setActiveTab: (tab: 'tabular' | 'newsletter') => void;
  onRunBatch: () => void;
  onOpenChat: () => void;
  onOpenLiveVoice: () => void;
  onOpenSettings: () => void;
  isRunning: boolean;
  itemCount: number;
}

export const Header: React.FC<HeaderProps> = ({
  activeTab,
  setActiveTab,
  onRunBatch,
  onOpenChat,
  onOpenLiveVoice,
  onOpenSettings,
  isRunning,
  itemCount,
}) => {
  return (
    <header className="border-b border-slate-800/80 bg-slate-950/80 backdrop-blur-xl sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          
          {/* Brand Logo & Title */}
          <div className="flex items-center gap-3">
            <div className="h-11 w-11 rounded-xl bg-gradient-to-tr from-indigo-600 via-purple-600 to-pink-500 p-0.5 shadow-lg shadow-indigo-500/20">
              <div className="h-full w-full bg-slate-950 rounded-[10px] flex items-center justify-center">
                <Bot className="h-6 w-6 text-indigo-400" />
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-bold tracking-tight text-white">AI Daily Brief</h1>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                  Gemini 3.7 & 5-Stream
                </span>
              </div>
              <p className="text-xs text-slate-400 mt-0.5">
                Frontier Models &bull; Google Cloud &bull; Research Papers &bull; Business &bull; OSS Tooling
              </p>
            </div>
          </div>

          {/* Quick Metrics & Actions */}
          <div className="flex items-center gap-2.5">
            <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-900/60 border border-slate-800 text-xs">
              <span className="text-slate-500">Indexed: </span>
              <span className="font-bold text-indigo-400">{itemCount}</span>
            </div>

            {/* Gemini Live Bidi Voice Button */}
            <button
              onClick={onOpenLiveVoice}
              className="flex items-center gap-1.5 px-3.5 py-2 rounded-lg font-semibold text-xs bg-purple-600/20 hover:bg-purple-600/30 text-purple-300 border border-purple-500/40 shadow-md shadow-purple-600/20 transition-all active:scale-95"
              title="Start Gemini Multimodal Live Bidi Voice Stream"
            >
              <Radio className="h-4 w-4 text-purple-400 animate-pulse" />
              <span>Live Voice</span>
            </button>

            {/* AI Assistant Chat Trigger */}
            <button
              onClick={onOpenChat}
              className="flex items-center gap-1.5 px-3.5 py-2 rounded-lg font-semibold text-xs bg-indigo-600/15 hover:bg-indigo-600/25 text-indigo-300 border border-indigo-500/30 transition-all active:scale-95"
            >
              <Sparkles className="h-4 w-4 text-indigo-400" />
              <span>AI Chat</span>
            </button>

            {/* Agent Settings Button */}
            <button
              onClick={onOpenSettings}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg font-semibold text-xs bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-300 transition-all active:scale-95"
              title="Agent & Gemini Model Settings"
            >
              <SettingsIcon className="h-4 w-4 text-slate-400" />
              <span className="hidden sm:inline">Settings</span>
            </button>

            <button
              onClick={onRunBatch}
              disabled={isRunning}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg font-semibold text-sm transition-all shadow-md ${
                isRunning
                  ? 'bg-slate-800 text-slate-400 cursor-not-allowed border border-slate-700'
                  : 'bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white shadow-indigo-600/25 hover:shadow-indigo-600/40 active:scale-95'
              }`}
            >
              {isRunning ? (
                <>
                  <RefreshCw className="h-4 w-4 animate-spin text-indigo-400" />
                  Crawling...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 fill-white" />
                  Run Batch Agent
                </>
              )}
            </button>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center gap-2 mt-6 border-t border-slate-800/60 pt-3">
          <button
            onClick={() => setActiveTab('tabular')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'tabular'
                ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/30 font-semibold'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/40'
            }`}
          >
            <Table className="h-4 w-4" />
            Tabular Intelligence Feed
          </button>

          <button
            onClick={() => setActiveTab('newsletter')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'newsletter'
                ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/30 font-semibold'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/40'
            }`}
          >
            <Mail className="h-4 w-4" />
            Daily Executive Digest
          </button>
        </div>
      </div>
    </header>
  );
};
