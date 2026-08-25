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

import { executeBatchRun } from '../server/agent/runner';

async function main() {
  console.log('----------------------------------------------------');
  console.log('🤖 AI Model News & Research Agent - Batch Runner');
  console.log('----------------------------------------------------');
  const result = await executeBatchRun();
  console.log(`Run Date:            ${result.run_date}`);
  console.log(`Status:              ${result.status.toUpperCase()}`);
  console.log(`New Non-Repeated:    ${result.new_items_inserted}`);
  console.log(`Duplicates Skipped:  ${result.skipped_duplicates}`);
  console.log(`Total DB Articles:   ${result.total_in_db}`);
  console.log('----------------------------------------------------');
  console.log('Execution Logs:');
  console.log(result.log);
  console.log('----------------------------------------------------');
}

main().catch((err) => {
  console.error('Fatal CLI batch run error:', err);
  process.exit(1);
});
