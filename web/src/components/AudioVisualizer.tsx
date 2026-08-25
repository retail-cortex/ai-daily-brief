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
import { Mic, Volume2, Sparkles } from 'lucide-react';

interface AudioVisualizerProps {
  state: 'idle' | 'listening' | 'reasoning' | 'speaking';
  transcript?: string;
  onStop?: () => void;
}

export const AudioVisualizer: React.FC<AudioVisualizerProps> = ({ state, transcript, onStop }) => {
  if (state === 'idle') return null;

  return (
    <div className="bg-slate-950/90 border border-indigo-500/40 rounded-2xl p-4 shadow-xl mb-3 animate-in fade-in zoom-in-95 duration-200">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          
          {/* Pulsing Icon */}
          <div
            className={`h-9 w-9 rounded-xl flex items-center justify-center shadow-lg transition-all ${
              state === 'listening'
                ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/40 animate-pulse'
                : state === 'reasoning'
                ? 'bg-indigo-500/20 text-indigo-400 border border-indigo-500/40 animate-spin'
                : 'bg-purple-500/20 text-purple-400 border border-purple-500/40 animate-pulse'
            }`}
          >
            {state === 'listening' && <Mic className="h-5 w-5" />}
            {state === 'reasoning' && <Sparkles className="h-5 w-5" />}
            {state === 'speaking' && <Volume2 className="h-5 w-5" />}
          </div>

          <div>
            <div className="flex items-center gap-2">
              <span className="text-xs font-bold text-slate-100 uppercase tracking-wider">
                {state === 'listening' && 'Listening to Speech...'}
                {state === 'reasoning' && 'Gemini Reasoning...'}
                {state === 'speaking' && 'Speaking Response...'}
              </span>
              <span
                className={`h-2 w-2 rounded-full ${
                  state === 'listening'
                    ? 'bg-emerald-400 animate-ping'
                    : state === 'reasoning'
                    ? 'bg-indigo-400 animate-pulse'
                    : 'bg-purple-400 animate-pulse'
                }`}
              />
            </div>
            <p className="text-[11px] text-slate-400 truncate max-w-xs mt-0.5">
              {transcript || (state === 'listening' ? 'Speak now into your microphone...' : 'Generating natural spoken voice...')}
            </p>
          </div>
        </div>

        {/* Animated Waveform Equalizer */}
        <div className="flex items-center gap-1 h-6 px-3 py-1 bg-slate-900/80 rounded-lg border border-slate-800">
          <span className={`w-1 rounded-full ${state === 'listening' ? 'bg-emerald-400 animate-bounce' : state === 'speaking' ? 'bg-purple-400 animate-bounce' : 'bg-indigo-400 animate-pulse'} h-2`} style={{ animationDelay: '0ms' }} />
          <span className={`w-1 rounded-full ${state === 'listening' ? 'bg-emerald-400 animate-bounce' : state === 'speaking' ? 'bg-purple-400 animate-bounce' : 'bg-indigo-400 animate-pulse'} h-4`} style={{ animationDelay: '150ms' }} />
          <span className={`w-1 rounded-full ${state === 'listening' ? 'bg-emerald-400 animate-bounce' : state === 'speaking' ? 'bg-purple-400 animate-bounce' : 'bg-indigo-400 animate-pulse'} h-6`} style={{ animationDelay: '300ms' }} />
          <span className={`w-1 rounded-full ${state === 'listening' ? 'bg-emerald-400 animate-bounce' : state === 'speaking' ? 'bg-purple-400 animate-bounce' : 'bg-indigo-400 animate-pulse'} h-3`} style={{ animationDelay: '450ms' }} />
          <span className={`w-1 rounded-full ${state === 'listening' ? 'bg-emerald-400 animate-bounce' : state === 'speaking' ? 'bg-purple-400 animate-bounce' : 'bg-indigo-400 animate-pulse'} h-5`} style={{ animationDelay: '200ms' }} />
        </div>

        {onStop && (
          <button
            onClick={onStop}
            className="px-2.5 py-1 text-[11px] font-semibold bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg border border-slate-700 transition-all"
          >
            Stop
          </button>
        )}
      </div>
    </div>
  );
};
