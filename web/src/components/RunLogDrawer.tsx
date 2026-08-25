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
import { Terminal, CheckCircle2, AlertCircle, X } from 'lucide-react';

interface RunLogDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  runResult: any;
  isRunning: boolean;
}

export const RunLogDrawer: React.FC<RunLogDrawerProps> = ({
  isOpen,
  onClose,
  runResult,
  isRunning,
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-b-0 bottom-0 left-0 right-0 z-50 bg-slate-950/95 border-t border-slate-800 shadow-2xl p-6 backdrop-blur-xl transition-all">
      <div className="max-w-7xl mx-auto space-y-4">
        
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Terminal className="h-5 w-5 text-indigo-400" />
            <h3 className="text-base font-bold text-slate-100">Batch Crawler Execution Terminal</h3>
            {isRunning ? (
              <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/30 animate-pulse">
                Crawling Google, Anthropic, OpenAI, X AI, arXiv...
              </span>
            ) : runResult?.status === 'success' ? (
              <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center gap-1">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Crawl Completed ({runResult.new_items_inserted ?? 0} new non-repeated items added)
              </span>
            ) : (
              <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-red-500/10 text-red-400 border border-red-500/30 flex items-center gap-1">
                <AlertCircle className="h-3.5 w-3.5" />
                Crawl Error
              </span>
            )}
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white p-1">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 font-mono text-xs text-slate-300 max-h-60 overflow-y-auto leading-relaxed shadow-inner">
          {isRunning ? (
            <div className="space-y-1 text-slate-400">
              <p className="text-indigo-400">[INIT] Spawning multi-source HTTP feed readers...</p>
              <p>[CRAWLER] Connecting to Google AI Blog RSS...</p>
              <p>[CRAWLER] Connecting to Anthropic Newsroom...</p>
              <p>[CRAWLER] Connecting to OpenAI Index...</p>
              <p>[CRAWLER] Connecting to xAI Announcements...</p>
              <p>[CRAWLER] Querying arXiv API (cat:cs.CL OR cat:cs.AI)...</p>
              <p>[CRAWLER] Querying Hugging Face Daily Papers API...</p>
              <p className="text-amber-400 animate-pulse">[PROCESSING] Running text cleaner & summarizer engine...</p>
            </div>
          ) : runResult ? (
            <pre className="whitespace-pre-wrap">{runResult.log || JSON.stringify(runResult, null, 2)}</pre>
          ) : (
            <p className="text-slate-500">Ready to initiate batch crawl run.</p>
          )}
        </div>

      </div>
    </div>
  );
};
