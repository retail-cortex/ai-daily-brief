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
import { Bot, Send, X, Trash2, ExternalLink, Globe, User, Mic, MicOff, Volume2, VolumeX, Radio, Sparkles } from 'lucide-react';
import type { NewsItem } from './TabularView';
import { MarkdownRenderer } from './MarkdownRenderer';
import { AudioVisualizer } from './AudioVisualizer';

const API_BASE = typeof window !== 'undefined' && window.location.port === '5173'
  ? 'http://localhost:3001/api'
  : '/api';

interface Message {
  id: string;
  role: 'user' | 'model';
  content: string;
  article_title?: string;
  article_url?: string;
  created_at?: string;
}

interface AgentChatDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  selectedArticle?: NewsItem | null;
  onClearSelectedArticle?: () => void;
}

function cleanMarkdownForSpeech(md: string): string {
	return md
		.replace(/```[\s\S]*?```/g, 'Code block omitted.')
		.replace(/`([^`]+)`/g, '$1')
		.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '$1')
		.replace(/[*#_~]/g, '')
		.replace(/---/g, '')
		.replace(/>/g, '')
		.trim();
}

export const AgentChatDrawer: React.FC<AgentChatDrawerProps> = ({
  isOpen,
  onClose,
  selectedArticle,
  onClearSelectedArticle,
}) => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState<string>('');
  const [isSending, setIsSending] = useState<boolean>(false);
  const [sessionId] = useState<string>(() => `sess_${Date.now()}`);
  
  // Voice & Speech states
  const [isLiveVoiceMode, setIsLiveVoiceMode] = useState<boolean>(false);
  const [isListening, setIsListening] = useState<boolean>(false);
  const [isSpeaking, setIsSpeaking] = useState<boolean>(false);
  const [speakingMessageId, setSpeakingMessageId] = useState<string | null>(null);
  const [liveTranscript, setLiveTranscript] = useState<string>('');
  const [speechSupported, setSpeechSupported] = useState<boolean>(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const recognitionRef = useRef<any>(null);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
      if (SpeechRecognition) {
        setSpeechSupported(true);
        const recognition = new SpeechRecognition();
        recognition.continuous = false;
        recognition.interimResults = true;
        recognition.lang = 'en-US';

        recognition.onstart = () => {
          setIsListening(true);
        };

        recognition.onresult = (event: any) => {
          let currentTranscript = '';
          for (let i = event.resultIndex; i < event.results.length; i++) {
            currentTranscript += event.results[i][0].transcript;
          }
          setLiveTranscript(currentTranscript);
        };

        recognition.onerror = (event: any) => {
          console.warn('Speech recognition error:', event.error);
          setIsListening(false);
        };

        recognition.onend = () => {
          setIsListening(false);
          setLiveTranscript((latest) => {
            if (latest.trim()) {
              handleSendMessage(latest.trim());
            }
            return '';
          });
        };

        recognitionRef.current = recognition;
      }
    }

    return () => {
      stopSpeaking();
      if (recognitionRef.current) {
        try {
          recognitionRef.current.stop();
        } catch {
          // ignore
        }
      }
    };
  }, []);

  useEffect(() => {
    if (isOpen) {
      loadHistory();
    } else {
      stopSpeaking();
      if (isListening && recognitionRef.current) {
        recognitionRef.current.stop();
      }
    }
  }, [isOpen]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, liveTranscript, isListening, isSpeaking]);

  const loadHistory = async () => {
    try {
      const res = await fetch(`${API_BASE}/agent/history?session_id=${sessionId}`);
      const data = await res.json();
      if (data.success && data.history && data.history.length > 0) {
        setMessages(data.history);
      } else {
        setMessages([
          {
            id: 'welcome',
            role: 'model',
            content: `👋 Hello! I am your **Google ADK & Gemini 3.7 Intelligence Agent**.

I have real-time access to our indexed news feed across **Frontier Models, Google Cloud, AI Research Papers, Business Deals, and OSS Tooling**.

🎙️ **Live Voice Mode is ready!** You can speak your questions or click any message to listen to spoken answers in real-time.`,
          },
        ]);
      }
    } catch (err) {
      console.error('Failed to load chat history:', err);
    }
  };

  const toggleListening = () => {
    if (!speechSupported) {
      alert('Speech recognition is not supported in this browser. Please use Chrome, Edge, or Safari.');
      return;
    }

    if (isListening) {
      recognitionRef.current?.stop();
    } else {
      stopSpeaking();
      setLiveTranscript('');
      try {
        recognitionRef.current?.start();
      } catch (err) {
        console.error('Failed to start speech recognition:', err);
      }
    }
  };

  const speakText = (text: string, messageId: string) => {
    if (typeof window === 'undefined' || !('speechSynthesis' in window)) return;

    window.speechSynthesis.cancel();

    if (isSpeaking && speakingMessageId === messageId) {
      setIsSpeaking(false);
      setSpeakingMessageId(null);
      return;
    }

    const cleanText = cleanMarkdownForSpeech(text);
    if (!cleanText) return;

    const utterance = new SpeechSynthesisUtterance(cleanText);
    utterance.rate = 1.05;
    utterance.pitch = 1.0;

    // Pick English natural voice if available
    const voices = window.speechSynthesis.getVoices();
    const englishVoice = voices.find((v) => v.lang.startsWith('en') && (v.name.includes('Google') || v.name.includes('Natural') || v.name.includes('Samantha')));
    if (englishVoice) {
      utterance.voice = englishVoice;
    }

    utterance.onstart = () => {
      setIsSpeaking(true);
      setSpeakingMessageId(messageId);
    };

    utterance.onend = () => {
      setIsSpeaking(false);
      setSpeakingMessageId(null);
    };

    utterance.onerror = () => {
      setIsSpeaking(false);
      setSpeakingMessageId(null);
    };

    window.speechSynthesis.speak(utterance);
  };

  const stopSpeaking = () => {
    if (typeof window !== 'undefined' && 'speechSynthesis' in window) {
      window.speechSynthesis.cancel();
    }
    setIsSpeaking(false);
    setSpeakingMessageId(null);
  };

  const handleSendMessage = async (customPrompt?: string) => {
    const textToSend = customPrompt || input;
    if (!textToSend.trim() || isSending) return;

    stopSpeaking();

    const userMsg: Message = {
      id: `user_${Date.now()}`,
      role: 'user',
      content: textToSend,
      article_title: selectedArticle?.title,
      article_url: selectedArticle?.link,
    };

    setMessages((prev) => [...prev, userMsg]);
    if (!customPrompt) setInput('');
    setIsSending(true);

    try {
      const res = await fetch(`${API_BASE}/agent/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: sessionId,
          message: textToSend,
          article_id: selectedArticle?.id || '',
        }),
      });
      const data = await res.json();
      if (data.success && data.result) {
        const modelMsgId = `model_${Date.now()}`;
        const modelMsg: Message = {
          id: modelMsgId,
          role: 'model',
          content: data.result.response,
          article_title: data.result.article_title,
          article_url: data.result.article_url,
        };
        setMessages((prev) => [...prev, modelMsg]);

        // If in Live Voice Mode or initiated by speech, auto-read response
        if (isLiveVoiceMode || customPrompt) {
          speakText(data.result.response, modelMsgId);
        }
      } else {
        setMessages((prev) => [
          ...prev,
          {
            id: `err_${Date.now()}`,
            role: 'model',
            content: `⚠️ Failed to get response: ${data.error || 'Unknown error'}. Check your Agent Settings.`,
          },
        ]);
      }
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          id: `err_${Date.now()}`,
          role: 'model',
          content: `⚠️ Network error: ${String(err)}`,
        },
      ]);
    } finally {
      setIsSending(false);
    }
  };

  const handleClearHistory = async () => {
    stopSpeaking();
    try {
      await fetch(`${API_BASE}/agent/history?session_id=${sessionId}`, { method: 'DELETE' });
      setMessages([]);
      loadHistory();
    } catch (err) {
      console.error('Failed to clear history:', err);
    }
  };

  const visualizerState = isListening ? 'listening' : isSending ? 'reasoning' : isSpeaking ? 'speaking' : 'idle';

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden bg-slate-950/60 backdrop-blur-sm flex justify-end animate-in fade-in duration-200">
      <div className="w-full max-w-xl bg-slate-900 border-l border-slate-800 h-full flex flex-col shadow-2xl animate-in slide-in-from-right duration-300">
        
        {/* Drawer Header */}
        <div className="p-4 bg-slate-950/90 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-indigo-600 via-purple-600 to-pink-500 p-0.5 shadow-md shadow-indigo-600/30">
              <div className="h-full w-full bg-slate-950 rounded-[9px] flex items-center justify-center">
                <Bot className="h-5 w-5 text-indigo-400" />
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-bold text-slate-100 text-sm">Gemini 3.7 Live Agent</h3>
                <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
              </div>
              <p className="text-[11px] text-slate-400">Context-Grounded Voice & Chat Assistant</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            
            {/* Live Voice Toggle */}
            <button
              onClick={() => {
                const nextState = !isLiveVoiceMode;
                setIsLiveVoiceMode(nextState);
                if (!nextState) stopSpeaking();
              }}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold border transition-all ${
                isLiveVoiceMode
                  ? 'bg-purple-600/20 text-purple-300 border-purple-500/40 shadow-md shadow-purple-600/20'
                  : 'bg-slate-900 text-slate-400 border-slate-800 hover:text-slate-200'
              }`}
              title="Toggle Live Voice Mode (Auto-reads responses and enables continuous hands-free dialogue)"
            >
              <Radio className={`h-3.5 w-3.5 ${isLiveVoiceMode ? 'text-purple-400 animate-pulse' : ''}`} />
              <span className="hidden sm:inline">Live Voice</span>
            </button>

            <button
              onClick={handleClearHistory}
              title="Clear Session History"
              className="p-2 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-all"
            >
              <Trash2 className="h-4 w-4" />
            </button>
            <button
              onClick={onClose}
              className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-all"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Grounding Context Banner */}
        {selectedArticle ? (
          <div className="bg-indigo-950/40 border-b border-indigo-500/20 px-4 py-2.5 flex items-center justify-between gap-3 text-xs">
            <div className="flex items-center gap-2 overflow-hidden">
              <Globe className="h-4 w-4 text-indigo-400 shrink-0" />
              <div className="truncate">
                <span className="text-indigo-300 font-semibold">Article Context: </span>
                <span className="text-slate-200">{selectedArticle.title}</span>
              </div>
            </div>
            {onClearSelectedArticle && (
              <button
                onClick={onClearSelectedArticle}
                className="text-slate-400 hover:text-slate-200 shrink-0 px-2 py-0.5 rounded bg-slate-900 border border-slate-800 text-[10px]"
                title="Switch to Full Newsletter Base Context"
              >
                Switch to Full Newsletter
              </button>
            )}
          </div>
        ) : (
          <div className="bg-indigo-950/30 border-b border-indigo-500/20 px-4 py-2 flex items-center justify-between gap-3 text-xs">
            <div className="flex items-center gap-2 overflow-hidden">
              <Sparkles className="h-3.5 w-3.5 text-indigo-400 shrink-0" />
              <div className="truncate">
                <span className="text-indigo-300 font-semibold">Base Context: </span>
                <span className="text-slate-300">⚡ Daily AI & Cloud Intelligence Digest (Full Newsletter & TL;DR)</span>
              </div>
            </div>
            <span className="px-2 py-0.5 rounded bg-indigo-900/40 text-[10px] text-indigo-300 border border-indigo-500/30 shrink-0">
              5 Streams
            </span>
          </div>
        )}

        {/* Messages Stream */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 text-sm leading-relaxed">
          
          {/* Active Audio Waveform Equalizer */}
          <AudioVisualizer
            state={visualizerState}
            transcript={liveTranscript}
            onStop={() => {
              if (isListening) recognitionRef.current?.stop();
              if (isSpeaking) stopSpeaking();
            }}
          />

          {messages.map((m) => (
            <div
              key={m.id}
              className={`flex gap-3 ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              {m.role === 'model' && (
                <div className="h-7 w-7 rounded-lg bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center shrink-0 mt-0.5">
                  <Bot className="h-4 w-4 text-indigo-400" />
                </div>
              )}

              <div
                className={`max-w-[85%] rounded-2xl px-4 py-3 text-xs space-y-2 leading-relaxed shadow-sm ${
                  m.role === 'user'
                    ? 'bg-indigo-600 text-white rounded-tr-none'
                    : 'bg-slate-950 border border-slate-800 text-slate-200 rounded-tl-none'
                }`}
              >
                {m.article_title && m.role === 'user' && (
                  <div className="text-[10px] bg-indigo-700/60 px-2 py-0.5 rounded text-indigo-200 truncate mb-1">
                    Context: {m.article_title}
                  </div>
                )}

                {m.role === 'model' ? (
                  <MarkdownRenderer content={m.content} />
                ) : (
                  <div className="whitespace-pre-wrap font-sans break-words">{m.content}</div>
                )}

                {/* Footer Actions on Model Bubbles */}
                {m.role === 'model' && (
                  <div className="pt-2 mt-2 border-t border-slate-800/80 flex items-center justify-between gap-2 text-[11px] text-slate-400">
                    <div className="flex items-center gap-2">
                      {/* Text-to-Speech Play Button */}
                      <button
                        onClick={() => speakText(m.content, m.id)}
                        className={`flex items-center gap-1 px-2 py-1 rounded-md text-[10px] font-semibold border transition-all ${
                          speakingMessageId === m.id
                            ? 'bg-purple-600/20 text-purple-300 border-purple-500/40'
                            : 'bg-slate-900 text-slate-400 hover:text-slate-200 border-slate-800'
                        }`}
                        title="Listen to this response"
                      >
                        {speakingMessageId === m.id ? (
                          <>
                            <VolumeX className="h-3 w-3 text-purple-400" />
                            <span>Stop</span>
                          </>
                        ) : (
                          <>
                            <Volume2 className="h-3 w-3 text-indigo-400" />
                            <span>Speak</span>
                          </>
                        )}
                      </button>
                    </div>

                    {m.article_url && (
                      <a
                        href={m.article_url}
                        target="_blank"
                        rel="noreferrer"
                        className="hover:underline flex items-center gap-1 text-indigo-400 truncate"
                      >
                        <Globe className="h-3 w-3" />
                        Source
                        <ExternalLink className="h-2.5 w-2.5" />
                      </a>
                    )}
                  </div>
                )}
              </div>

              {m.role === 'user' && (
                <div className="h-7 w-7 rounded-lg bg-slate-800 border border-slate-700 flex items-center justify-center shrink-0 mt-0.5">
                  <User className="h-4 w-4 text-slate-300" />
                </div>
              )}
            </div>
          ))}

          <div ref={messagesEndRef} />
        </div>

        {/* Quick Prompt Suggestions */}
        <div className="px-4 py-2 bg-slate-950/60 border-t border-slate-800/60 flex items-center gap-1.5 overflow-x-auto no-scrollbar">
          <span className="text-[10px] font-semibold text-slate-500 whitespace-nowrap mr-1">Quick Prompts:</span>
          {selectedArticle ? (
            <>
              <button
                onClick={() => handleSendMessage('Summarize the key technical takeaways and benchmarks from this article.')}
                className="px-2.5 py-1 rounded-lg text-[11px] bg-slate-900 border border-slate-800 text-slate-300 hover:border-indigo-500 hover:text-white whitespace-nowrap transition-all"
              >
                ⚡ Key Takeaways
              </button>
              <button
                onClick={() => handleSendMessage('How does this release compare to competitors in latency, pricing, and performance?')}
                className="px-2.5 py-1 rounded-lg text-[11px] bg-slate-900 border border-slate-800 text-slate-300 hover:border-indigo-500 hover:text-white whitespace-nowrap transition-all"
              >
                📊 Compare
              </button>
            </>
          ) : (
            <>
              <button
                onClick={() => handleSendMessage('What are the top 3 AI model releases and breakthroughs this week?')}
                className="px-2.5 py-1 rounded-lg text-[11px] bg-slate-900 border border-slate-800 text-slate-300 hover:border-indigo-500 hover:text-white whitespace-nowrap transition-all"
              >
                🔥 Top Releases
              </button>
              <button
                onClick={() => handleSendMessage('What are the latest major updates in Google Cloud & Vertex AI release notes?')}
                className="px-2.5 py-1 rounded-lg text-[11px] bg-slate-900 border border-slate-800 text-slate-300 hover:border-indigo-500 hover:text-white whitespace-nowrap transition-all"
              >
                ☁️ Google Cloud
              </button>
            </>
          )}
        </div>

        {/* Input Bar with Microphone Voice Action */}
        <div className="p-4 bg-slate-950 border-t border-slate-800">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleSendMessage();
            }}
            className="flex items-center gap-2"
          >
            {/* Microphone Button */}
            <button
              type="button"
              onClick={toggleListening}
              className={`p-2.5 rounded-xl border transition-all ${
                isListening
                  ? 'bg-emerald-500 text-slate-950 border-emerald-400 shadow-lg shadow-emerald-500/30 animate-pulse'
                  : 'bg-slate-900 text-slate-300 border-slate-800 hover:text-white hover:border-indigo-500'
              }`}
              title={isListening ? 'Stop listening' : 'Speak to Gemini Agent (Microphone)'}
            >
              {isListening ? <MicOff className="h-4 w-4" /> : <Mic className="h-4 w-4 text-indigo-400" />}
            </button>

            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={isListening ? 'Listening to speech...' : selectedArticle ? `Ask about "${selectedArticle.title.substring(0, 30)}..."` : 'Type or speak your question...'}
              className="flex-1 bg-slate-900 border border-slate-800 rounded-xl px-4 py-2.5 text-xs text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-all"
            />

            <button
              type="submit"
              disabled={!input.trim() || isSending}
              className="p-2.5 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white shadow-lg shadow-indigo-600/30 transition-all disabled:opacity-50"
            >
              <Send className="h-4 w-4" />
            </button>
          </form>
        </div>
      </div>
    </div>
  );
};
