import React from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../button';
import { useDarkMode } from '../../contexts/DarkModeContext';

interface ButtonProps {
  text: string;
  onClick?: () => void;
  variant?: 'primary' | 'secondary' | 'outline';
  size?: 'sm' | 'md' | 'lg';
}

interface NavbarProps {
  title: React.ReactNode;
  leftButtons?: ButtonProps[];
  rightButtons?: ButtonProps[];
}

const Navbar: React.FC<NavbarProps> = ({
  title,
  leftButtons = [],
  rightButtons = []
}) => {
  const navigate = useNavigate();
  const { isDarkMode, toggleDarkMode } = useDarkMode();

  const handleTitleClick = () => {
    navigate('/');
  };

  return (
    <nav className="fixed top-0 left-0 right-0 bg-sky-50 dark:bg-slate-900 shadow-sm z-50 transition-colors">
      <div className="container mx-auto px-4 py-3 flex items-center justify-between">
        <div className="flex space-x-2">
          {leftButtons.map((button, index) => (
            <Button
              key={index}
              text={button.text}
              onClick={button.onClick}
              variant={button.variant || 'outline'}
              size="sm"
            />
          ))}
        </div>

        <h1
          className="text-xl font-semibold text-sky-700 dark:text-sky-300 cursor-pointer hover:text-sky-800 dark:hover:text-sky-200 transition-colors"
          onClick={handleTitleClick}
        >
          {title}
        </h1>

        <div className="flex space-x-2 items-center">
          <button
            onClick={toggleDarkMode}
            className="p-2 rounded-lg bg-gray-200 dark:bg-slate-700 text-gray-800 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-slate-600 transition-colors"
            aria-label="Toggle dark mode"
            title={isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {isDarkMode ? (
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                <circle cx="12" cy="12" r="5" />
                <path d="M12 1v6M12 17v6M23 12h-6M7 12H1M20.485 3.515l-4.243 4.243M7.758 16.242l-4.243 4.243M20.485 20.485l-4.243-4.243M7.758 7.758L3.515 3.515" />
              </svg>
            )}
          </button>
          {rightButtons.map((button, index) => (
            <Button
              key={index}
              text={button.text}
              onClick={button.onClick}
              variant={button.variant || 'outline'}
              size="sm"
            />
          ))}
        </div>
      </div>
    </nav>
  );
};

export default Navbar;