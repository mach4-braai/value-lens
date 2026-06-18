import { useState } from "react";

interface Props {
  onSubmit: (question: string) => void;
  disabled: boolean;
}

export function QueryInput({ onSubmit, disabled }: Props) {
  const [question, setQuestion] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (question.trim()) {
      onSubmit(question.trim());
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ marginBottom: 24 }}>
      <textarea
        value={question}
        onChange={(e) => setQuestion(e.target.value)}
        placeholder="e.g. What are Tesla's biggest risk factors?"
        rows={3}
        style={{ width: "100%", padding: 12, fontSize: 16 }}
        disabled={disabled}
      />
      <button
        type="submit"
        disabled={disabled || !question.trim()}
        style={{ marginTop: 8, padding: "8px 24px", fontSize: 16 }}
      >
        Ask
      </button>
    </form>
  );
}
