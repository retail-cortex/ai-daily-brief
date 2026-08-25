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

import React, { useState, useEffect, useRef } from 'react';
import { Mic, MicOff, PhoneOff, Globe, Sparkles, Volume2, Radio, AlertCircle, Send } from 'lucide-react';
import type { NewsItem } from './TabularView';
import { PCMAudioRecorder, PCMAudioPlayer } from '../utils/pcmAudio';

interface GeminiLiveVoiceModalProps {
  isOpen: boolean;
  onClose: () => void;
  selectedArticle?: NewsItem | null;
  onClearSelectedArticle?: () => void;
}

const NEURAL_VOICES = [
  { id: 'Aoede', name: 'Aoede', desc: 'Warm & Expressive (Default)' },
  { id: 'Puck', name: 'Puck', desc: 'Upbeat & Energetic' },
  { id: 'Charon', name: 'Charon', desc: 'Calm & Authoritative' },
  { id: 'Fenrir', name: 'Fenrir', desc: 'Bold & Dynamic' },
  { id: 'Kore', name: 'Kore', desc: 'Gentle & Relaxed' },
];

export const GeminiLiveVoiceModal: React.FC<GeminiLiveVoiceModalProps> = ({
  isOpen,
  onClose,
  selectedArticle,
  onClearSelectedArticle,
}) => {
  const [selectedVoice, setSelectedVoice] = useState<string>('Aoede');
  const [connectionStatus, setConnectionStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected');
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [isMicMuted, setIsMicMuted] = useState<boolean>(false);
  const [agentState, setAgentState] = useState<'idle' | 'listening' | 'speaking' | 'reasoning'>('idle');
  const [transcript, setTranscript] = useState<{ role: 'user' | 'model'; text: string }[]>([]);
  const [userVolume, setUserVolume] = useState<number>(0);
  const [modelVolume, setModelVolume] = useState<number>(0);
  const [textInput, setTextInput] = useState<string>('');

  const wsRef = useRef<WebSocket | null>(null);
  const recorderRef = useRef<PCMAudioRecorder | null>(null);
  const playerRef = useRef<PCMAudioPlayer | null>(null);
  const animationFrameRef = useRef<number | null>(null);

  useEffect(() => {
    if (isOpen) {
      startLiveSession();
    } else {
      endLiveSession();
    }

    return () => {
      endLiveSession();
    };
  }, [isOpen, selectedVoice, selectedArticle]);

  const startLiveSession = async () => {
    endLiveSession();
    setConnectionStatus('connecting');
    setErrorMessage('');
    setTranscript([]);

    try {
      // 1. Initialize 24kHz PCM Player
      playerRef.current = new PCMAudioPlayer();

      // 2. Establish WebSocket connection to backend Bidi Live proxy
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.port === '5173' ? 'localhost:3001' : window.location.host;
      const articleParam = selectedArticle?.id ? `&article_id=${encodeURIComponent(selectedArticle.id)}` : '';
      const wsUrl = `${proto}//${host}/api/agent/live?voice=${encodeURIComponent(selectedVoice)}${articleParam}`;

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = async () => {
        setConnectionStatus('connected');
        setAgentState('listening');

        // 3. Start Microphone Capture (16kHz PCM)
        try {
          recorderRef.current = new PCMAudioRecorder((base64Chunk) => {
            if (ws.readyState === WebSocket.OPEN && !isMicMuted) {
              ws.send(
                JSON.stringify({
                  realtimeInput: {
                    mediaChunks: [
                      {
                        mimeType: 'audio/pcm;rate=16000',
                        data: base64Chunk,
                      },
                    ],
                  },
                })
              );
            }
          });

          await recorderRef.current.start();
          startVolumeMonitoring();
        } catch (micErr) {
          console.warn('[Bidi Live] Microphone access notice:', micErr);
          // Mic permission may be pending or unavailable, user can still type or hear Gemini
        }
      };

      ws.onmessage = async (event) => {
        try {
          let textData = '';
          if (typeof event.data === 'string') {
            textData = event.data;
          } else if (event.data instanceof Blob) {
            textData = await event.data.text();
          } else if (event.data instanceof ArrayBuffer) {
            textData = new TextDecoder().decode(event.data);
          }

          const data = JSON.parse(textData);

          if (data.error) {
            setConnectionStatus('error');
            setErrorMessage(data.error);
            return;
          }

          if (data.connected || data.setupComplete) {
            setConnectionStatus('connected');
            setAgentState('listening');
          }

          // Handle serverContent from Gemini Live
          if (data.serverContent) {
            const modelTurn = data.serverContent.modelTurn;
            if (modelTurn && modelTurn.parts) {
              setAgentState('speaking');
              for (const part of modelTurn.parts) {
                // Incoming 24kHz PCM Audio Chunk
                if (part.inlineData && part.inlineData.data) {
                  playerRef.current?.playBase64Chunk(part.inlineData.data);
                }
                // Incoming text transcript
                if (part.text) {
                  setTranscript((prev) => {
                    const last = prev[prev.length - 1];
                    if (last && last.role === 'model') {
                      return [...prev.slice(0, -1), { role: 'model', text: last.text + part.text }];
                    }
                    return [...prev, { role: 'model', text: part.text }];
                  });
                }
              }
            }

            if (data.serverContent.turnComplete) {
              setAgentState('listening');
            }

            if (data.serverContent.interrupted) {
              playerRef.current?.interrupt();
              setAgentState('listening');
            }
          }
        } catch (err) {
          console.error('[Bidi Live] Message decode error:', err);
        }
      };

      ws.onerror = (err) => {
        console.error('[Bidi Live] WebSocket error:', err);
        setConnectionStatus('error');
        setErrorMessage('Failed to connect to Gemini Live Multimodal Service.');
      };

      ws.onclose = (event) => {
        if (event.code !== 1000 && event.code !== 1005) {
          setConnectionStatus('error');
          setErrorMessage(event.reason || 'Gemini Live stream closed unexpectedly.');
        } else {
          setConnectionStatus('disconnected');
          setAgentState('idle');
        }
      };
    } catch (err) {
      console.error('[Bidi Live] Error starting session:', err);
      setConnectionStatus('error');
      setErrorMessage(String(err));
    }
  };

  const handleSendTextMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!textInput.trim() || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;

    playerRef.current?.interrupt();
    setAgentState('reasoning');

    setTranscript((prev) => [...prev, { role: 'user', text: textInput.trim() }]);

    wsRef.current.send(
      JSON.stringify({
        clientContent: {
          turns: [
            {
              role: 'user',
              parts: [{ text: textInput.trim() }],
            },
          ],
          turnComplete: true,
        },
      })
    );

    setTextInput('');
  };

  const startVolumeMonitoring = () => {
    const updateVolumes = () => {
      if (recorderRef.current) {
        setUserVolume(recorderRef.current.getVolume());
      }
      if (playerRef.current) {
        setModelVolume(playerRef.current.getVolume());
      }
      animationFrameRef.current = requestAnimationFrame(updateVolumes);
    };
    updateVolumes();
  };

  const endLiveSession = () => {
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }
    if (recorderRef.current) {
      recorderRef.current.stop();
      recorderRef.current = null;
    }
    if (playerRef.current) {
      playerRef.current.close();
      playerRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnectionStatus('disconnected');
    setAgentState('idle');
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/85 backdrop-blur-xl flex items-center justify-center p-4 animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-3xl max-w-2xl w-full p-6 sm:p-8 space-y-6 shadow-2xl relative overflow-hidden flex flex-col max-h-[90vh]">
        
        {/* Top Header Banner */}
        <div className="flex items-center justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-3">
            <div className="h-11 w-11 rounded-2xl bg-gradient-to-tr from-purple-600 via-indigo-600 to-pink-500 p-0.5 shadow-lg shadow-purple-600/30">
              <div className="h-full w-full bg-slate-950 rounded-[14px] flex items-center justify-center">
                <Radio className="h-5 w-5 text-purple-400 animate-pulse" />
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-bold text-white tracking-tight">Gemini Live Bidi Voice</h2>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-purple-500/20 text-purple-300 border border-purple-500/40 flex items-center gap-1">
                  <span className="h-1.5 w-1.5 rounded-full bg-purple-400 animate-ping" />
                  ADK Bidi Stream
                </span>
              </div>
              <p className="text-xs text-slate-400 mt-0.5">Google Multimodal Live API &bull; 24kHz Native Neural Audio</p>
            </div>
          </div>

          {/* Voice Selector */}
          <div className="flex items-center gap-2">
            <select
              value={selectedVoice}
              onChange={(e) => setSelectedVoice(e.target.value)}
              className="bg-slate-950 border border-slate-800 rounded-xl px-3 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-purple-500 cursor-pointer"
            >
              {NEURAL_VOICES.map((v) => (
                <option key={v.id} value={v.id}>
                  🎙️ {v.name} ({v.desc})
                </option>
              ))}
            </select>

            <button
              onClick={onClose}
              className="text-slate-400 hover:text-white p-2 rounded-xl hover:bg-slate-800 transition-all text-sm font-semibold"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Active Grounded Context Banner */}
        {selectedArticle ? (
          <div className="bg-indigo-950/40 border border-indigo-500/30 rounded-2xl px-4 py-3 flex items-center justify-between gap-3 text-xs">
            <div className="flex items-center gap-2.5 overflow-hidden">
              <Globe className="h-4 w-4 text-indigo-400 shrink-0" />
              <div className="truncate">
                <span className="text-indigo-300 font-bold">Article Context: </span>
                <span className="text-slate-100">{selectedArticle.title}</span>
              </div>
            </div>
            {onClearSelectedArticle && (
              <button
                onClick={onClearSelectedArticle}
                className="text-slate-400 hover:text-slate-200 shrink-0 px-2 py-0.5 rounded bg-slate-900 border border-slate-800 text-[11px]"
                title="Switch to Full Newsletter Base Context"
              >
                Switch to Full Newsletter
              </button>
            )}
          </div>
        ) : (
          <div className="bg-purple-950/30 border border-purple-500/30 rounded-2xl px-4 py-3 flex items-center justify-between gap-3 text-xs">
            <div className="flex items-center gap-2.5 overflow-hidden">
              <Sparkles className="h-4 w-4 text-purple-400 shrink-0" />
              <div className="truncate">
                <span className="text-purple-300 font-bold">Base Context: </span>
                <span className="text-slate-200">⚡ Daily AI & Cloud Intelligence Digest (Full Newsletter & TL;DR)</span>
              </div>
            </div>
            <span className="px-2 py-0.5 rounded bg-purple-900/40 text-[10px] text-purple-300 border border-purple-500/30 shrink-0">
              5 Streams Included
            </span>
          </div>
        )}

        {/* Central Audio Waveform & Visualizer Stage */}
        <div className="flex-1 min-h-[200px] bg-slate-950/80 rounded-2xl border border-slate-800/80 p-6 flex flex-col items-center justify-center relative overflow-hidden">
          
          {/* Background Ambient Glow */}
          <div
            className={`absolute inset-0 transition-opacity duration-700 blur-3xl pointer-events-none ${
              agentState === 'speaking'
                ? 'bg-purple-600/15 opacity-100'
                : agentState === 'listening'
                ? 'bg-emerald-600/10 opacity-100'
                : 'opacity-0'
            }`}
          />

          {/* Central Pulsing Orb */}
          <div className="relative mb-4">
            <div
              className={`h-24 w-24 rounded-full flex items-center justify-center transition-all duration-300 shadow-2xl ${
                connectionStatus === 'connecting'
                  ? 'bg-slate-800 text-slate-400 animate-pulse'
                  : agentState === 'speaking'
                  ? 'bg-gradient-to-tr from-purple-600 to-pink-500 text-white scale-110 shadow-purple-600/40'
                  : agentState === 'listening'
                  ? 'bg-gradient-to-tr from-emerald-600 to-teal-500 text-white scale-100 shadow-emerald-600/40'
                  : 'bg-slate-800 text-slate-300'
              }`}
              style={{
                transform: agentState === 'speaking'
                  ? `scale(${1.05 + Math.min(modelVolume * 2.5, 0.4)})`
                  : agentState === 'listening'
                  ? `scale(${1.0 + Math.min(userVolume * 3.0, 0.35)})`
                  : 'scale(1)',
              }}
            >
              {connectionStatus === 'connecting' ? (
                <Sparkles className="h-8 w-8 animate-spin" />
              ) : agentState === 'speaking' ? (
                <Volume2 className="h-8 w-8 animate-pulse" />
              ) : (
                <Mic className="h-8 w-8" />
              )}
            </div>

            {/* Orbit Ring */}
            <div
              className={`absolute -inset-3 rounded-full border border-dashed transition-all duration-500 ${
                agentState === 'speaking'
                  ? 'border-purple-400/50 animate-spin'
                  : agentState === 'listening'
                  ? 'border-emerald-400/50 animate-spin'
                  : 'border-slate-800'
              }`}
              style={{ animationDuration: '8s' }}
            />
          </div>

          {/* Status Text Indicator */}
          <div className="text-center space-y-1 z-10">
            <div className="text-sm font-bold text-slate-100 flex items-center justify-center gap-2">
              {connectionStatus === 'connecting' && 'Connecting to Google Gemini Live API...'}
              {connectionStatus === 'connected' && agentState === 'listening' && 'Listening... Speak naturally anytime'}
              {connectionStatus === 'connected' && agentState === 'speaking' && `Gemini is Speaking (${selectedVoice})...`}
              {connectionStatus === 'connected' && agentState === 'reasoning' && 'Processing Live Multimodal Context...'}
              {connectionStatus === 'error' && 'Connection Error'}
            </div>
            <p className="text-xs text-slate-400">
              {connectionStatus === 'connected'
                ? 'Full-duplex stream active &bull; Speak or type below to converse'
                : errorMessage || 'Establishing real-time bidirectional stream...'}
            </p>
          </div>

          {/* Real-Time Live Transcript Preview */}
          {transcript.length > 0 && (
            <div className="mt-4 w-full max-h-24 overflow-y-auto px-4 py-2 bg-slate-900/90 rounded-xl border border-slate-800 text-xs text-slate-300 text-center leading-relaxed">
              <span className="text-purple-300 font-semibold">{transcript[transcript.length - 1].role === 'model' ? 'Gemini: ' : 'You: '}</span>
              {transcript[transcript.length - 1].text}
            </div>
          )}
        </div>

        {/* Error Feedback */}
        {connectionStatus === 'error' && (
          <div className="p-3.5 rounded-2xl bg-red-500/10 border border-red-500/30 text-red-300 text-xs flex items-start gap-2.5">
            <AlertCircle className="h-4 w-4 text-red-400 shrink-0 mt-0.5" />
            <div className="flex-1 break-words">{errorMessage}</div>
          </div>
        )}

        {/* Quick Text Input for Hybrid Voice & Text interaction */}
        <form onSubmit={handleSendTextMessage} className="flex items-center gap-2">
          <input
            type="text"
            value={textInput}
            onChange={(e) => setTextInput(e.target.value)}
            placeholder="Type a message or speak into your microphone..."
            className="flex-1 bg-slate-950 border border-slate-800 rounded-xl px-4 py-2.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:border-purple-500 transition-all"
          />
          <button
            type="submit"
            disabled={!textInput.trim()}
            className="p-2.5 rounded-xl bg-purple-600 hover:bg-purple-500 text-white transition-all disabled:opacity-40"
          >
            <Send className="h-4 w-4" />
          </button>
        </form>

        {/* Bottom Call Controls */}
        <div className="flex items-center justify-center gap-4 pt-1">
          
          {/* Mute Mic Button */}
          <button
            onClick={() => setIsMicMuted(!isMicMuted)}
            className={`p-3.5 rounded-2xl border transition-all shadow-lg ${
              isMicMuted
                ? 'bg-red-500/20 border-red-500/40 text-red-300'
                : 'bg-slate-950 border-slate-800 text-slate-200 hover:border-slate-700'
            }`}
            title={isMicMuted ? 'Unmute Microphone' : 'Mute Microphone'}
          >
            {isMicMuted ? <MicOff className="h-5 w-5 text-red-400" /> : <Mic className="h-5 w-5 text-emerald-400" />}
          </button>

          {/* End Live Call Button */}
          <button
            onClick={onClose}
            className="flex items-center gap-2 px-6 py-3.5 rounded-2xl font-semibold text-sm bg-red-600 hover:bg-red-500 text-white shadow-xl shadow-red-600/30 transition-all active:scale-95"
          >
            <PhoneOff className="h-5 w-5" />
            <span>End Live Voice Session</span>
          </button>
        </div>
      </div>
    </div>
  );
};
