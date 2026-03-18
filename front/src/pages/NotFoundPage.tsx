import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <div className="container">
      <div className="card">
        <h1 style={{ fontSize: '48px', marginBottom: '16px' }}>404</h1>
        <h2 style={{ marginBottom: '16px' }}>Page Not Found</h2>
        <p className="subtitle" style={{ marginBottom: '24px' }}>
          The page you're looking for doesn't exist or has been removed.
        </p>
        <Link to="/" className="btn">
          Go to Home
        </Link>
      </div>
    </div>
  );
}
