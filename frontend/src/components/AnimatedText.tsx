import { useState } from "react";

const AnimatedText = ({ children }) => {
  const [hovered, setHovered] = useState(false);

  const glintStyle = {
    backgroundSize: "200% auto",
    animation: "glint 2s linear infinite",
    WebkitBackgroundClip: "text",
    WebkitTextFillColor: "transparent",
    transition: "opacity 0.8s ease-in-out",
  };

  return (
    <>
      <style>
        {`
          @keyframes glint {
            0% { background-position: -200% 0; }
            100% { background-position: 200% 0; }
          }
        `}
      </style>

      <h1
        className="text-5xl font-bold leading-[1.2] inline-block select-none text-white transition-colors duration-500"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      >
        {hovered ? (
          <span
            className="bg-gradient-to-r from-purple-400 via-pink-500 to-blue-400 bg-clip-text"
            style={glintStyle}
          >
            {children}
          </span>
        ) : (
          children
        )}
      </h1>
    </>
  );
};

export default AnimatedText;
