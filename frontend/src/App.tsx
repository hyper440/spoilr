import { useState, useEffect } from 'react'
import {GreetService} from "@bindings/changeme";
import {Events, WML} from "@wailsio/runtime";

import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

function App() {
  const [name, setName] = useState<string>('');
  const [result, setResult] = useState<string>('Please enter your name below 👇');
  const [time, setTime] = useState<string>('Listening for Time event...');

  const doGreet = () => {
    let localName = name;
    if (!localName) {
      localName = 'anonymous';
    }
    GreetService.Greet(localName).then((resultValue: string) => {
      setResult(resultValue);
    }).catch((err: any) => {
      console.log(err);
    });
  }

  useEffect(() => {
    Events.On('time', (timeValue: any) => {
      setTime(timeValue.data);
    });
    // Reload WML so it picks up the wml tags
    WML.Reload();
  }, []);

  return (
    <div className="container bg-background">
      <div>
        <a data-wml-openURL="https://wails.io">
          <img src="/wails.png" className="logo" alt="Wails logo"/>
        </a>
        <a data-wml-openURL="https://reactjs.org">
          <img src="/react.svg" className="logo react" alt="React logo"/>
        </a>
      </div>
      <h1>Wails + React</h1>
      <div className="result">{result}</div>
      <div className="card ">
        <div className="input-box">
          <Input className="input" value={name} onChange={(e) => setName(e.target.value)} type="text" autoComplete="off"/>
          <Button className="btn" onClick={doGreet}>Greet</Button>
        </div>
      </div>
      <div className="footer">
        <div><p>Click on the Wails logo to learn more</p></div>
        <div><p>{time}</p></div>
      </div>
    </div>
  )
}

export default App
