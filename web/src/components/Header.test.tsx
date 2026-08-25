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
