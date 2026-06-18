interface Source {
  ticker: string;
  filed_date: string;
  section: string;
  excerpt: string;
  distance: number;
}

interface Props {
  result: {
    answer: string;
    sources: Source[];
  };
}

export function ResultCard({ result }: Props) {
  return (
    <div>
      <h2>Answer</h2>
      <div style={{ whiteSpace: "pre-wrap", lineHeight: 1.6 }}>
        {result.answer}
      </div>

      <h3 style={{ marginTop: 24 }}>Sources ({result.sources.length})</h3>
      {result.sources.map((s, i) => (
        <div
          key={i}
          style={{
            border: "1px solid #333",
            borderRadius: 8,
            padding: 12,
            marginBottom: 8,
          }}
        >
          <strong>
            {s.ticker} | {s.filed_date} | {s.section}
          </strong>
          <span style={{ float: "right", color: "#888" }}>
            dist: {s.distance.toFixed(4)}
          </span>
          <p style={{ marginTop: 8, color: "#ccc" }}>{s.excerpt}</p>
        </div>
      ))}
    </div>
  );
}
