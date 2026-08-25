import { describe, test, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import { Header } from './Header';
import * as matchers from '@testing-library/jest-dom/matchers';
expect.extend(matchers);

describe('Header Component', () => {
  const defaultProps = {
    activeTab: 'tabular' as const,
    setActiveTab: vi.fn(),
    onRunBatch: vi.fn(),
    onOpenChat: vi.fn(),
    onOpenLiveVoice: vi.fn(),
    onOpenSettings: vi.fn(),
    isRunning: false,
    itemCount: 42,
  };

  test('renders Header with title and metrics count', () => {
    render(<Header {...defaultProps} />);
    
    // Check main title
    expect(screen.getByText('AI Daily Brief')).toBeInTheDocument();
    
    // Check metric item count
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  test('fires callbacks on click events', () => {
    render(<Header {...defaultProps} />);
    
    // Click Run Batch button
    const runBtn = screen.getByText('Run Batch Agent');
    fireEvent.click(runBtn);
    expect(defaultProps.onRunBatch).toHaveBeenCalled();

    // Click Live Voice button
    const liveVoiceBtn = screen.getByText('Live Voice');
    fireEvent.click(liveVoiceBtn);
    expect(defaultProps.onOpenLiveVoice).toHaveBeenCalled();

    // Click AI Chat button
    const chatBtn = screen.getByText('AI Chat');
    fireEvent.click(chatBtn);
    expect(defaultProps.onOpenChat).toHaveBeenCalled();

    // Click Settings button
    const settingsBtn = screen.getByText('Settings');
    fireEvent.click(settingsBtn);
    expect(defaultProps.onOpenSettings).toHaveBeenCalled();
  });

  test('displays crawling state when isRunning is true', () => {
    render(<Header {...defaultProps} isRunning={true} />);
    expect(screen.getByText('Crawling...')).toBeInTheDocument();
    expect(screen.queryByText('Run Batch Agent')).not.toBeInTheDocument();
  });
});
