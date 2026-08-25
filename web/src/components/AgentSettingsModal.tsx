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
import { Bot, Key, Cloud, CheckCircle2, AlertCircle, RefreshCw, X, Sparkles } from 'lucide-react';

const API_BASE = typeof window !== 'undefined' && window.location.port === '5173'
  ? 'http://localhost:3001/api'
  : '/api';

interface AgentSettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AgentSettingsModal: React.FC<AgentSettingsModalProps> = ({ isOpen, onClose }) => {
  const [model, setModel] = useState<string>('gemini-3.7-flash');
  const [customModel, setCustomModel] = useState<string>('');
  const [authMode, setAuthMode] = useState<'api_key' | 'vertex_adc'>('api_key');
  const [apiKey, setApiKey] = useState<string>('');
  const [projectId, setProjectId] = useState<string>('');
  const [location, setLocation] = useState<string>('us-central1');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isTesting, setIsTesting] = useState<boolean>(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [saveSuccess, setSaveSuccess] = useState<boolean>(false);

  const presetModels = [
    { id: 'gemini-3.7-flash', name: 'Gemini 3.7 Flash', desc: 'Frontier Agent & High-Speed Coding (Recommended)' },
    { id: 'gemini-3.1-pro', name: 'Gemini 3.1 Pro', desc: 'Deep Reasoning & Architecture Flagship' },
    { id: 'gemini-3.5-flash', name: 'Gemini 3.5 Flash', desc: 'Fast Agentic Workflows' },
    { id: 'gemini-3.5-flash-lite', name: 'Gemini 3.5 Flash-Lite', desc: 'High-Throughput Efficiency' },
    { id: 'gemini-2.5-flash', name: 'Gemini 2.5 Flash', desc: 'Stable Baseline Tier' },
    { id: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro', desc: 'Deep Context Baseline' },
    { id: 'custom', name: 'Custom Model ID...', desc: 'Enter custom or preview model string' },
  ];

  useEffect(() => {
    if (isOpen) {
      loadSettings();
    }
  }, [isOpen]);

  const loadSettings = async () => {
    setIsLoading(true);
    setTestResult(null);
    setSaveSuccess(false);
    try {
      const res = await fetch(`${API_BASE}/settings`);
      const data = await res.json();
      if (data.success && data.settings) {
        const s = data.settings;
        const currentModel = s.gemini_model || 'gemini-3.7-flash';
        const isPreset = presetModels.some((m) => m.id === currentModel);
        if (isPreset) {
          setModel(currentModel);
        } else {
          setModel('custom');
          setCustomModel(currentModel);
        }
        setAuthMode(s.gemini_auth_mode === 'vertex_adc' ? 'vertex_adc' : 'api_key');
        setApiKey(s.gemini_api_key || '');
        setProjectId(s.vertex_project_id || '');
        setLocation(s.vertex_location || 'us-central1');
      }
    } catch (err) {
      console.error('Failed to load settings:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    setIsLoading(true);
    setSaveSuccess(false);
    setTestResult(null);

    const chosenModel = model === 'custom' ? customModel.trim() || 'gemini-3.7-flash' : model;

    try {
      const res = await fetch(`${API_BASE}/settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gemini_model: chosenModel,
          gemini_auth_mode: authMode,
          gemini_api_key: apiKey.trim(),
          vertex_project_id: projectId.trim(),
          vertex_location: location.trim(),
        }),
      });
      const data = await res.json();
      if (data.success) {
        setSaveSuccess(true);
        setTimeout(() => setSaveSuccess(false), 3000);
      }
    } catch (err) {
      console.error('Failed to save settings:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleTestConnection = async () => {
    setIsTesting(true);
    setTestResult(null);
    await handleSave();

    const chosenModel = model === 'custom' ? customModel.trim() || 'gemini-3.7-flash' : model;

    try {
      const res = await fetch(`${API_BASE}/agent/test-connection?model=${encodeURIComponent(chosenModel)}`, {
        method: 'POST',
      });
      const data = await res.json();
      if (data.success) {
        setTestResult({ success: true, message: `Connected to ${chosenModel} successfully!` });
      } else {
        setTestResult({ success: false, message: data.error || 'Connection failed' });
      }
    } catch (err) {
      setTestResult({ success: false, message: 'Network error: ' + String(err) });
    } finally {
      setIsTesting(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-xl w-full p-6 space-y-6 shadow-2xl relative animate-in fade-in zoom-in-95 duration-200">
        
        {/* Modal Header */}
        <div className="flex items-start justify-between border-b border-slate-800 pb-4">
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-gradient-to-tr from-indigo-600 to-purple-600 p-0.5 shadow-md shadow-indigo-600/30">
              <div className="h-full w-full bg-slate-950 rounded-[10px] flex items-center justify-center">
                <Bot className="h-5 w-5 text-indigo-400" />
              </div>
            </div>
            <div>
              <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                Agent & Model Settings
                <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
                  Gemini 3.7
                </span>
              </h2>
              <p className="text-xs text-slate-400">Configure LLM intelligence engine, model selection, and cloud credentials.</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-all"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Model Selection */}
        <div className="space-y-2">
          <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5 text-indigo-400" />
            Gemini Model Engine
          </label>
          <select
            value={model}
            onChange={(e) => setModel(e.target.value)}
            className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 cursor-pointer"
          >
            {presetModels.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name} &bull; {m.desc}
              </option>
            ))}
          </select>
          {model === 'custom' && (
            <input
              type="text"
              value={customModel}
              onChange={(e) => setCustomModel(e.target.value)}
              placeholder="e.g. gemini-3.7-flash-preview"
              className="w-full mt-2 bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500"
            />
          )}
        </div>

        {/* Authentication Mode */}
        <div className="space-y-3">
          <label className="text-xs font-semibold text-slate-300">Authentication Method</label>
          <div className="grid grid-cols-2 gap-3">
            <button
              type="button"
              onClick={() => setAuthMode('api_key')}
              className={`p-3 rounded-xl border flex flex-col items-start text-left transition-all ${
                authMode === 'api_key'
                  ? 'bg-indigo-600/15 border-indigo-500 text-slate-100 shadow-md shadow-indigo-600/20'
                  : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'
              }`}
            >
              <div className="flex items-center gap-1.5 font-semibold text-xs text-indigo-300 mb-1">
                <Key className="h-3.5 w-3.5" />
                Google AI Studio Key
              </div>
              <span className="text-[11px] text-slate-400">Direct API key from ai.google.dev</span>
            </button>

            <button
              type="button"
              onClick={() => setAuthMode('vertex_adc')}
              className={`p-3 rounded-xl border flex flex-col items-start text-left transition-all ${
                authMode === 'vertex_adc'
                  ? 'bg-indigo-600/15 border-indigo-500 text-slate-100 shadow-md shadow-indigo-600/20'
                  : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'
              }`}
            >
              <div className="flex items-center gap-1.5 font-semibold text-xs text-sky-300 mb-1">
                <Cloud className="h-3.5 w-3.5" />
                Vertex AI (ADC)
              </div>
              <span className="text-[11px] text-slate-400">Google Cloud Default Credentials</span>
            </button>
          </div>
        </div>

        {/* Auth Mode Inputs */}
        {authMode === 'api_key' ? (
          <div className="space-y-2 bg-slate-950/60 p-4 rounded-xl border border-slate-800">
            <label className="text-xs font-semibold text-slate-300 flex items-center justify-between">
              <span>Google Gemini API Key</span>
              <a
                href="https://aistudio.google.com/app/apikey"
                target="_blank"
                rel="noreferrer"
                className="text-[11px] text-indigo-400 hover:underline"
              >
                Get API Key &rarr;
              </a>
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="AIzaSy..."
              className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 font-mono"
            />
          </div>
        ) : (
          <div className="space-y-3 bg-slate-950/60 p-4 rounded-xl border border-slate-800">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-semibold text-slate-300 block mb-1">GCP Project ID</label>
                <input
                  type="text"
                  value={projectId}
                  onChange={(e) => setProjectId(e.target.value)}
                  placeholder="e.g. my-project-id (or auto-detected from ADC)"
                  className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 font-mono"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-slate-300 block mb-1">GCP Location / Region</label>
                <input
                  type="text"
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  placeholder="us-central1"
                  className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 font-mono"
                />
              </div>
            </div>
            <p className="text-[11px] text-slate-400 leading-relaxed">
              ⚡ <strong>Zero API key required.</strong> Automatically uses Application Default Credentials (ADC) from <code>gcloud auth application-default login</code> or GCP compute metadata.
            </p>
          </div>
        )}

        {/* Test Result Feedback */}
        {testResult && (
          <div
            className={`p-3.5 rounded-xl border flex items-start gap-2.5 text-xs ${
              testResult.success
                ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-300'
                : 'bg-red-500/10 border-red-500/30 text-red-300'
            }`}
          >
            {testResult.success ? (
              <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0 mt-0.5" />
            ) : (
              <AlertCircle className="h-4 w-4 text-red-400 shrink-0 mt-0.5" />
            )}
            <div className="flex-1 break-words">{testResult.message}</div>
          </div>
        )}

        {saveSuccess && (
          <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
            Agent settings saved successfully!
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex items-center justify-between pt-2 border-t border-slate-800">
          <button
            type="button"
            onClick={handleTestConnection}
            disabled={isTesting}
            className="px-4 py-2 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-all flex items-center gap-1.5 disabled:opacity-50"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isTesting ? 'animate-spin' : ''}`} />
            {isTesting ? 'Testing Connection...' : 'Test Connection'}
          </button>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-400 hover:text-slate-200"
            >
              Close
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={isLoading}
              className="px-5 py-2 rounded-xl text-xs font-semibold bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white shadow-lg shadow-indigo-600/30 transition-all disabled:opacity-50"
            >
              {isLoading ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
