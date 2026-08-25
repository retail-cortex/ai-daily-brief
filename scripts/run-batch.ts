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
