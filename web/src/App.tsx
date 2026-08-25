import { useState, useEffect } from 'react';
import { Header } from './components/Header';
import { TabularView, type NewsItem } from './components/TabularView';
import { NewsletterView } from './components/NewsletterView';
import { RunLogDrawer } from './components/RunLogDrawer';
import { AgentChatDrawer } from './components/AgentChatDrawer';
import { AgentSettingsModal } from './components/AgentSettingsModal';
import { GeminiLiveVoiceModal } from './components/GeminiLiveVoiceModal';

const API_BASE = typeof window !== 'undefined' && window.location.port === '5173'
  ? 'http://localhost:3001/api'
  : '/api';

export function App() {
  const [activeTab, setActiveTab] = useState<'tabular' | 'newsletter'>('tabular');
  const [items, setItems] = useState<NewsItem[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isRunning, setIsRunning] = useState<boolean>(false);
  const [showLogDrawer, setShowLogDrawer] = useState<boolean>(false);
  const [lastRunResult, setLastRunResult] = useState<any>(null);

  // Agent Chat, Live Voice & Settings states
  const [isChatOpen, setIsChatOpen] = useState<boolean>(false);
  const [isLiveVoiceOpen, setIsLiveVoiceOpen] = useState<boolean>(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState<boolean>(false);
  const [selectedArticle, setSelectedArticle] = useState<NewsItem | null>(null);

  const fetchItems = async () => {
    setIsLoading(true);
    try {
      const res = await fetch(`${API_BASE}/items`);
      const data = await res.json();
      if (data.success) {
        setItems(data.items);
      }
    } catch (err) {
      console.error('Error loading news items:', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchItems();
  }, []);

  const handleRunBatch = async () => {
    setIsRunning(true);
    setShowLogDrawer(true);
    setLastRunResult(null);

    try {
      const res = await fetch(`${API_BASE}/batch/run`, { method: 'POST' });
      const data = await res.json();
      if (data.success) {
        setLastRunResult(data.result);
        await fetchItems();
      } else {
        setLastRunResult({ status: 'failed', log: data.error || 'Batch crawl failed' });
      }
    } catch (err) {
      setLastRunResult({ status: 'failed', log: 'Network error executing batch: ' + String(err) });
    } finally {
      setIsRunning(false);
    }
  };

  const handleOpenChatWithArticle = (item: NewsItem) => {
    setSelectedArticle(item);
    setIsChatOpen(true);
  };

  const handleOpenLiveVoiceWithArticle = (item: NewsItem) => {
    setSelectedArticle(item);
    setIsLiveVoiceOpen(true);
  };

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col selection:bg-indigo-500 selection:text-white">
      
      {/* Header Navigation */}
      <Header
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        onRunBatch={handleRunBatch}
        onOpenChat={() => setIsChatOpen(true)}
        onOpenLiveVoice={() => setIsLiveVoiceOpen(true)}
        onOpenSettings={() => setIsSettingsOpen(true)}
        isRunning={isRunning}
        itemCount={items.length}
      />

      {/* Main View Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'tabular' && (
          <TabularView
            items={items}
            isLoading={isLoading}
            onOpenChatWithArticle={handleOpenChatWithArticle}
            onOpenLiveVoiceWithArticle={handleOpenLiveVoiceWithArticle}
          />
        )}
        {activeTab === 'newsletter' && <NewsletterView />}
      </main>

      {/* Execution Log Drawer */}
      <RunLogDrawer
        isOpen={showLogDrawer}
        onClose={() => setShowLogDrawer(false)}
        runResult={lastRunResult}
        isRunning={isRunning}
      />

      {/* Interactive Agent Chat Drawer */}
      <AgentChatDrawer
        isOpen={isChatOpen}
        onClose={() => setIsChatOpen(false)}
        selectedArticle={selectedArticle}
        onClearSelectedArticle={() => setSelectedArticle(null)}
      />

      {/* Gemini Multimodal Live Bidi Voice Modal */}
      <GeminiLiveVoiceModal
        isOpen={isLiveVoiceOpen}
        onClose={() => setIsLiveVoiceOpen(false)}
        selectedArticle={selectedArticle}
        onClearSelectedArticle={() => setSelectedArticle(null)}
      />

      {/* Agent & Model Settings Modal */}
      <AgentSettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
      />

      {/* App Footer */}
      <footer className="border-t border-slate-800/80 bg-slate-950/40 py-6 mt-12">
        <div className="max-w-7xl mx-auto px-4 text-center text-xs text-slate-500">
          <p>AI Daily Brief &bull; Standalone Go + Gin + GORM + SQLite + React</p>
        </div>
      </footer>
    </div>
  );
}

export default App;
