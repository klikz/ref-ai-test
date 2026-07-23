import { Backend_Request } from "@/services/backend";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";

import { Input } from "@/components/ui/input"
import { ShowErrorToast } from "@/components/showToast";


export default function RejaPage() {

  const [count, setCount] = useState(0)


  // async function getReja() {
  //   const result = await Backend_Request({}, '/api/lines/reja')
  //   if 
  // }

    async function getReja() {
      let result = await Backend_Request({}, "/api/lines/reja")
      // console.log(result)
      if (result.result === "ok") {
        // setT1(result.data.t1)
        // setT2(result.data.t2)
        setCount(result.data)
      } else {
        ShowErrorToast(result.error);
      }
    }

    async function updateReja() {
      let result = await Backend_Request({
        count: Number(count)
      }, "/api/lines/reja/update")
      // console.log(result)
      if (result.result === "ok") {
        // setT1(result.data.t1)
        // setT2(result.data.t2)
        setCount(result.data)
      } else {
        ShowErrorToast(result.error);
      }
    }


  useEffect(() => {
    getReja()
  }, []);

  return (
    <div>
      <div style={{ marginTop: 10, marginLeft: 20 }}>
        Reja: 
        <Input
          style={{ width: 150, marginLeft: 5 }}
          value={count}
          inputMode="numeric"
          onChange={(e) => setCount(parseInt(e.target.value))}
          // ref={serialRef} id="input-demo-api-key" type="text" placeholder="Qayta chiqarish"

        />
        <Button onClick={updateReja} style={{marginLeft: 10}}>Yangilash</Button>
      </div>
    </div>
  );
}
