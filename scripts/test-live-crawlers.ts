import Parser from 'rss-parser';
import * as cheerio from 'cheerio';

const parser = new Parser({
  timeout: 8000,
  headers: {
    'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  },
});

async function testCrawlers() {
  console.log('--- TESTING LIVE CRAWLERS ---');

  // 1. Google AI / DeepMind
  try {
    const gFeed = await parser.parseURL('https://blog.google/technology/ai/rss/');
    console.log('Google AI RSS Items found:', gFeed.items.length);
    if (gFeed.items[0]) console.log('  Top item:', gFeed.items[0].title, '|', gFeed.items[0].link);
  } catch (e) {
    console.log('Google RSS Error:', e);
  }

  // 2. Anthropic HTML Scraper
  try {
    const res = await fetch('https://www.anthropic.com/news', {
      headers: { 'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' }
    });
    if (res.ok) {
      const html = await res.text();
      const $ = cheerio.load(html);
      const links: { title: string; link: string }[] = [];
      $('a[href*="/news/"]').each((_, el) => {
        const title = $(el).text().trim();
        const href = $(el).attr('href');
        if (title && href && title.length > 8 && !links.some(l => l.link === href)) {
          links.push({ title, link: href.startsWith('http') ? href : `https://www.anthropic.com${href}` });
        }
      });
      console.log('Anthropic Live HTML News found:', links.length);
      if (links[0]) console.log('  Top item:', links[0].title, '|', links[0].link);
    } else {
      console.log('Anthropic fetch status:', res.status);
    }
  } catch (e) {
    console.log('Anthropic Error:', e);
  }

  // 3. OpenAI HTML Scraper
  try {
    const res = await fetch('https://openai.com/news/', {
      headers: { 'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' }
    });
    if (res.ok) {
      const html = await res.text();
      const $ = cheerio.load(html);
      const links: { title: string; link: string }[] = [];
      $('a[href*="/index/"]').each((_, el) => {
        const title = $(el).text().trim();
        const href = $(el).attr('href');
        if (title && href && title.length > 8 && !links.some(l => l.link === href)) {
          links.push({ title, link: href.startsWith('http') ? href : `https://openai.com${href}` });
        }
      });
      console.log('OpenAI Live HTML News found:', links.length);
      if (links[0]) console.log('  Top item:', links[0].title, '|', links[0].link);
    }
  } catch (e) {
    console.log('OpenAI Error:', e);
  }

  // 4. X AI HTML Scraper
  try {
    const res = await fetch('https://x.ai/blog', {
      headers: { 'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' }
    });
    if (res.ok) {
      const html = await res.text();
      const $ = cheerio.load(html);
      const links: { title: string; link: string }[] = [];
      $('a[href*="/blog/"]').each((_, el) => {
        const title = $(el).text().trim();
        const href = $(el).attr('href');
        if (title && href && title.length > 5 && href !== '/blog' && href !== '/blog/' && !links.some(l => l.link === href)) {
          links.push({ title, link: href.startsWith('http') ? href : `https://x.ai${href}` });
        }
      });
      console.log('X AI Live HTML News found:', links.length);
      if (links[0]) console.log('  Top item:', links[0].title, '|', links[0].link);
    }
  } catch (e) {
    console.log('X AI Error:', e);
  }

  // 5. Hugging Face Papers
  try {
    const res = await fetch('https://huggingface.co/api/daily_papers?limit=5');
    if (res.ok) {
      const papers = await res.json();
      console.log('Hugging Face Daily Papers API found:', papers.length);
      if (papers[0]?.paper) console.log('  Top paper:', papers[0].paper.title, '| https://huggingface.co/papers/' + papers[0].paper.id);
    }
  } catch (e) {
    console.log('HF Error:', e);
  }

  // 6. arXiv API
  try {
    const res = await fetch('https://export.arxiv.org/api/query?search_query=cat:cs.CL+OR+cat:cs.AI&sortBy=submittedDate&sortOrder=descending&max_results=5');
    if (res.ok) {
      const xml = await res.text();
      const $ = cheerio.load(xml, { xmlMode: true });
      console.log('arXiv API Papers found:', $('entry').length);
      const firstTitle = $('entry').first().find('title').text().trim();
      console.log('  Top paper:', firstTitle);
    }
  } catch (e) {
    console.log('arXiv Error:', e);
  }
}

testCrawlers().catch(console.error);
