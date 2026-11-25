import os,json,subprocess,hashlib
from pathlib import Path
p=Path(__file__).parent;e=os.environ.copy();e.update(json.loads((p/'clone-runtime-env.json').read_text()))
c=subprocess.Popen([str(p/'app-api'),'-f',str(p/'clone-app.yaml')],env=e,cwd=p/'input/server',stdout=(p/'api-runtime.log').open('a'),stderr=subprocess.STDOUT,start_new_session=True)
(p/'api-process.json').write_text(json.dumps({'pid':c.pid,'source':'e9762f566a1c03bef99f42630f0d4384f2538443','binarySHA256':hashlib.sha256((p/'app-api').read_bytes()).hexdigest()}));print('Candidate API pid',c.pid)
