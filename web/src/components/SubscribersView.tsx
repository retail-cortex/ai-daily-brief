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
import { Users, UserPlus, Trash2 } from 'lucide-react';

export interface Subscriber {
  id: string;
  email: string;
  name: string;
  added_at?: string;
}

interface SubscribersViewProps {
  subscribers: Subscriber[];
  onAddSubscriber: (email: string, name: string) => void;
  onDeleteSubscriber: (id: string) => void;
}

export const SubscribersView: React.FC<SubscribersViewProps> = ({
  subscribers,
  onAddSubscriber,
  onDeleteSubscriber,
}) => {
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !email.includes('@')) {
      setError('Please enter a valid email address');
      return;
    }
    setError(null);
    onAddSubscriber(email, name);
    setEmail('');
    setName('');
  };

  return (
    <div className="space-y-6">
      
      {/* Top Header Card */}
      <div className="glass-panel p-6 rounded-2xl flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5 text-indigo-400" />
            <h2 className="text-lg font-bold text-slate-100">Daily Newsletter Subscriber List</h2>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Manage stakeholders, executive teams, and team distribution lists who will receive the daily AI briefing.
          </p>
        </div>
        <div className="px-4 py-2 rounded-xl bg-indigo-500/10 border border-indigo-500/30 text-indigo-300 font-semibold text-sm">
          Total Subscribers: {subscribers.length}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        
        {/* Add Subscriber Form */}
        <div className="glass-panel p-6 rounded-2xl space-y-4 h-fit border border-slate-800">
          <div className="flex items-center gap-2">
            <UserPlus className="h-4 w-4 text-emerald-400" />
            <h3 className="text-base font-bold text-slate-100">Add New Recipient</h3>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Subscriber Name / Group</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. AI Research Group"
                className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Email Address *</label>
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="e.g. team-leads@company.com"
                className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            {error && <div className="text-xs text-red-400 font-medium">{error}</div>}

            <button
              type="submit"
              className="w-full py-2.5 rounded-xl font-semibold text-sm bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/30 transition-all flex items-center justify-center gap-2"
            >
              <UserPlus className="h-4 w-4" />
              Add to Subscription List
            </button>
          </form>
        </div>

        {/* Subscriber List Table */}
        <div className="lg:col-span-2 glass-panel rounded-2xl overflow-hidden border border-slate-800">
          <div className="bg-slate-900 px-6 py-4 border-b border-slate-800 flex items-center justify-between">
            <h3 className="text-sm font-bold text-slate-200">Active Recipients ({subscribers.length})</h3>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="bg-slate-950/60 text-xs font-semibold text-slate-400 border-b border-slate-800">
                  <th className="py-3 px-6">Subscriber Name</th>
                  <th className="py-3 px-6">Email Address</th>
                  <th className="py-3 px-6">Added Date</th>
                  <th className="py-3 px-6 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {subscribers.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="text-center py-8 text-slate-400">
                      No subscribers currently added.
                    </td>
                  </tr>
                ) : (
                  subscribers.map((sub) => (
                    <tr key={sub.id} className="hover:bg-slate-800/40 transition-colors">
                      <td className="py-3.5 px-6 font-semibold text-slate-200">{sub.name}</td>
                      <td className="py-3.5 px-6 font-mono text-indigo-300 text-xs">{sub.email}</td>
                      <td className="py-3.5 px-6 text-slate-400 text-xs">
                        {sub.added_at ? sub.added_at.split('T')[0] : 'System Seed'}
                      </td>
                      <td className="py-3.5 px-6 text-right">
                        <button
                          onClick={() => onDeleteSubscriber(sub.id)}
                          className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition-all"
                          title="Remove subscriber"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

      </div>

    </div>
  );
};
