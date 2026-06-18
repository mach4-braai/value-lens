import { useState } from "react";
import { QueryInput } from "./components/QueryInput";
import { ResultCard } from "./components/ResultCard";

interface Source {
  ticker: string;
  filed_date: string;
  section: string;
  excerpt: string;
  distance: number;
}

interface QueryResult {
  answer: string;
  sources: Source[];
}

export default function App() {
  const [result, setResult] = useState<QueryResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleQuery(question: string) {
    setLoading(true);
    setError("");
    try {
      const resp = await fetch("/api/query", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question, top_k: 5 }),
      });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const data: QueryResult = await resp.json();
      setResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Query failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ maxWidth: 800, margin: "0 auto", padding: 24 }}>
      <h1>Value Lens</h1>
      <p style={{ color: "#888" }}>
        Ask questions about SEC filings for TSLA, AAPL, GOOGL, NFLX, META
      </p>
      <QueryInput onSubmit={handleQuery} disabled={loading} />
      {loading && <p>Querying...</p>}
      {error && <p style={{ color: "red" }}>{error}</p>}
      {result && <ResultCard result={result} />}
    </div>
  );
}
