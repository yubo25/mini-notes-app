import React, { useState, useEffect } from 'react';
import './App.css';
// 导入官方浏览器端 SDK 运行时
import { AnnaAppRuntime } from '@anna-ai/app-runtime';

async function connectHost() {
  return await AnnaAppRuntime.connect();
}

function App() {
  const [anna, setAnna] = useState(null);
  const [notes, setNotes] = useState([]);
  const [newNote, setNewNote] = useState('');
  const [summary, setSummary] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    connectHost()
      .then((client) => {
        setAnna(client);
        loadNotes(client);
      })
      .catch((err) => {
        console.error("handshake connection failed:", err);
        setError("Could not establish handshake with Anna Host.");
      });
  }, []);

  const loadNotes = async (client) => {
    try {
      const savedNotes = await client.storage.get({ key: "notes" });
      if (savedNotes && Array.isArray(savedNotes)) {
        setNotes(savedNotes);
      } else {
        setNotes([]);
      }
    } catch (err) {
      console.error("Storage load error:", err);
      setError("Failed to fetch notes from host storage.");
    }
  };

  const handleAddNote = async (e) => {
    e.preventDefault();
    if (!newNote.trim()) return;

    const updatedNotes = [
      ...notes,
      {
        id: Date.now().toString(),
        content: newNote.trim(),
        createdAt: new Date().toISOString()
      }
    ];

    try {
      setNotes(updatedNotes);
      setNewNote('');
      if (anna) {
        await anna.storage.set({ key: "notes", value: updatedNotes });
      }
    } catch (err) {
      console.error("Storage set error:", err);
      setError("Failed to sync new note with host storage.");
    }
  };

  const handleDeleteNote = async (id) => {
    const updatedNotes = notes.filter(n => n.id !== id);
    try {
      setNotes(updatedNotes);
      if (anna) {
        await anna.storage.set({ key: "notes", value: updatedNotes });
      }
    } catch (err) {
      console.error("Storage delete error:", err);
      setError("Failed to sync note removal with host storage.");
    }
  };
  const handleSummarize = async () => {
    if (notes.length === 0) {
      setError("Note list is empty.");
      return;
    }

    setLoading(true);
    setError(null);
    setSummary('');

    try {
      const texts = notes.map(n => n.content);

      const res = await anna.tools.invoke({
        tool_id: "note-summarizer.summarize",
        tool: "note-summarizer.summarize",
        name: "note-summarizer.summarize",
        args: { notes: texts },
        params: { notes: texts }
      });
      
      if (res && res.output) {
        setSummary(res.output);
      } else {
        setSummary(JSON.stringify(res));
      }
    } catch (err) {
      console.error("Invoke error:", err);
      setError(err.message || String(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="app-container">
      <header className="app-header">
        <h1>📝 Mini Notes</h1>
        <p className="app-meta">Local Notes & LLM Sampling Summarizer</p>
      </header>

      <div className="workspace">
        <section className="notes-pane">
          <h2>Create Note</h2>
          <form onSubmit={handleAddNote} className="add-form">
            <input
              type="text"
              placeholder="Enter some brief notes..."
              value={newNote}
              onChange={(e) => setNewNote(e.target.value)}
              className="add-input"
            />
            <button type="submit" className="add-btn">Save</button>
          </form>

          {notes.length === 0 ? (
            <div className="empty-message">No notes present. Enter content to persist.</div>
          ) : (
            <div className="notes-list-wrapper">
              <ul className="notes-list">
                {notes.map((note, index) => (
                  <li key={note.id} className="note-card">
                    <span className="note-seq">#{index + 1}</span>
                    <p className="note-text">{note.content}</p>
                    <button
                      onClick={() => handleDeleteNote(note.id)}
                      className="delete-btn"
                      title="Delete entry"
                    >
                      &times;
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </section>

        <section className="summary-pane">
          <h2>LLM Summary Integration</h2>
          <button
            onClick={handleSummarize}
            disabled={loading || notes.length === 0}
            className="summarize-btn"
          >
            {loading ? "Processing..." : "✨ Summarize Notes"}
          </button>

          {error && (
            <div className="error-alert">
              <strong>Host Return Value / Error:</strong>
              <pre className="error-code">{error}</pre>
            </div>
          )}

          {summary && (
            <div className="summary-card">
              <h3>Summarized Result:</h3>
              <div className="summary-body">{summary}</div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

export default App;