import React, { useState, useEffect } from 'react';
import { Settings as SettingsIcon, Save, Clock, Mail, CheckCircle2 } from 'lucide-react';

const API_BASE = typeof window !== 'undefined' && window.location.port === '5173'
  ? 'http://localhost:3001/api'
  : '/api';

export const SettingsModal: React.FC = () => {
  const [cronSchedule, setCronSchedule] = useState('0 8 * * *');
  const [savedSuccess, setSavedSuccess] = useState(false);

  useEffect(() => {
    fetch(`${API_BASE}/settings`)
      .then((res) => res.json())
      .then((data) => {
        if (data.success && data.settings) {
          setCronSchedule(data.settings.cron_schedule || '0 8 * * *');
        }
      })
      .catch(console.error);
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_BASE}/settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          cron_schedule: cronSchedule,
        }),
      });
      const data = await res.json();
      if (data.success) {
        setSavedSuccess(true);
        setTimeout(() => setSavedSuccess(false), 3000);
      }
    } catch (err) {
      alert('Error saving settings: ' + String(err));
    }
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      
      {/* Header Banner */}
      <div className="glass-panel p-6 rounded-2xl flex items-center justify-between border border-slate-800">
        <div>
          <div className="flex items-center gap-2">
            <SettingsIcon className="h-5 w-5 text-indigo-400" />
            <h2 className="text-lg font-bold text-slate-100">Agent & Scheduler Configuration</h2>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Set up your automated batch cron frequency and AI settings.
          </p>
        </div>
      </div>

      {savedSuccess && (
        <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-sm font-semibold flex items-center gap-2">
          <CheckCircle2 className="h-5 w-5 text-emerald-400" />
          Settings successfully updated and applied!
        </div>
      )}

      <form onSubmit={handleSave} className="space-y-6">

        {/* Cron Schedule Settings */}
        <div className="glass-panel p-6 rounded-2xl space-y-4 border border-slate-800">
          <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
            <Clock className="h-4 w-4 text-purple-400" />
            <h3 className="text-base font-bold text-slate-100">Automated Daily Batch Schedule</h3>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 mb-1">Cron Expression</label>
            <input
              type="text"
              value={cronSchedule}
              onChange={(e) => setCronSchedule(e.target.value)}
              placeholder="0 8 * * *"
              className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm text-slate-100 font-mono"
            />
            <p className="text-xs text-slate-400 mt-2">
              Default <code className="bg-slate-900 px-1.5 py-0.5 rounded text-indigo-300">0 8 * * *</code> executes every day at 8:00 AM local time.
            </p>
          </div>
        </div>

        {/* Save Button */}
        <div className="flex justify-end">
          <button
            type="submit"
            className="flex items-center gap-2 px-6 py-3 rounded-xl font-semibold text-sm bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white shadow-lg shadow-indigo-600/30 transition-all"
          >
            <Save className="h-4 w-4" />
            Save Configuration Settings
          </button>
        </div>

      </form>
    </div>
  );
};
