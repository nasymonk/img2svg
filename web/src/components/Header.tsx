import { Link, useLocation } from 'react-router-dom';

export default function Header() {
  const { pathname } = useLocation();

  return (
    <header className="bg-surface border-b border-border">
      <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link to="/" className="text-lg font-bold text-primary">
            img2svg
          </Link>
          <nav className="flex gap-4 text-sm">
            <Link
              to="/"
              className={`${pathname === '/' ? 'text-text' : 'text-text-secondary'} hover:text-text transition-colors`}
            >
              转换
            </Link>
            <Link
              to="/history"
              className={`${pathname === '/history' ? 'text-text' : 'text-text-secondary'} hover:text-text transition-colors`}
            >
              历史
            </Link>
          </nav>
        </div>
      </div>
    </header>
  );
}
