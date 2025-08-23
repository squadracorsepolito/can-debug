# CAN Debug Tool

Un tool di debug per messaggi CAN con interfaccia TUI (Text User Interface) basato su bubbletea di Charm.

## Funzionalità

- **Apertura file DBC**: Seleziona e carica un file .dbc dal filesystem usando il file picker integrato
- **Selezione multipla messaggi**: Scegli uno o più messaggi CAN da monitorare
- **Visualizzazione segnali DBC**: Mostra tutti i segnali di ogni messaggio selezionato con:
  - Nome del messaggio e ID CAN in formato esadecimale
  - Nome del segnale con unità di misura (se disponibili)
  - Valore del segnale in formato decimale
  - Posizione bit nel payload
  - Tipo di segnale (standard, enum, muxor) con dimensione in bit
- **Tabella navigabile**: Usa frecce su/giù per scorrere i segnali nella tabella di monitoraggio
- **Interfaccia TUI**: Usa bubbletea di Charm del terminale
- **Integrazione acmelib**: Utilizza acmelib per parsing di file DBC e strutture messaggi

## Installazione

```bash
cd cmd/can-debug
go build
```

## Utilizzo

### Modalità base (file picker)

```bash
./can-debug
```

### Caricamento diretto di un file DBC

```bash
./can-debug path/to/file.dbc
```

### Esempio con il file DBC

```bash
./can-debug ./internal/test/MCB.dbc
```

### Controlli Tastiera

#### File Picker

- **Frecce ↑/↓**: Navigazione file e directory
- **Enter**: Apri directory o seleziona file .dbc
- **Backspace**: Torna alla directory precedente

#### Selezione Messaggi

- **Frecce ↑/↓**: Naviga lista messaggi CAN
- **Space**: Toggle selezione/deselezione messaggio
- **Enter**: Inizia monitoraggio dei messaggi selezionati
- **/ (slash)**: Filtra messaggi per nome

#### Monitoraggio Segnali

- **Frecce ↑/↓**: Scorri la tabella dei segnali
- **Tab**: Torna alla selezione messaggi

#### Globali

- **q / Ctrl+C**: Esci dall'applicazione
- **?**: Mostra aiuto (dove disponibile)

### Flusso di utilizzo

1. **Selezione file DBC**: Naviga nel filesystem e seleziona un file .dbc
2. **Selezione messaggi**: Usa SPACE per selezionare/deselezionare i messaggi CAN che vuoi monitorare
3. **Visualizzazione segnali**: Visualizza tutti i segnali dei messaggi selezionati con:
   - Informazioni strutturali dal file DBC
   - Posizioni bit e tipi di segnale
   - Unità di misura quando disponibili
   - Possibilità di scorrere la tabella con le frecce

## Struttura del progetto

```
can-debug/
├── main.go                    # Entry point dell'applicazione
├── internal/
│   ├── test/                  # Test e file di esempio
│   │   └── MCB.dbc            # File DBC di esempio per test
│   ├── ui/                    # Package per l'interfaccia utente
│   │   ├── types.go           # Definizioni di tipi e strutture
│   │   ├── model.go           # Costruttori e inizializzazione
│   │   ├── update.go          # Logica di aggiornamento del modello
│   │   ├── view.go            # Logica di rendering delle viste
│   │   └── handlers.go        # Handler per eventi e logica business
│   └── can/                   # Package per la logica CAN
│       └── decoder.go         # Decoder per messaggi CAN usando acmelib
└── README.md
```

### Descrizione dei file

- **handlers.go**: Contiene tutta la logica business dell'applicazione:
  - `loadDBC()`: Carica e parsifica file DBC usando acmelib
  - `setupMessageList()`: Configura la lista dei messaggi CAN disponibili
  - `toggleMessageSelection()`: Gestisce la selezione/deselezione dei messaggi
  - `updateMessageList()`: Aggiorna la lista dei messaggi selezionati
  - `setupMonitoringTable()`: Configura la tabella di monitoraggio con focus abilitato
  - `initializesTableDBCSignals()`: Initializes (using only the selected signals) a table for visualizing the received messages
  - `getSignalTypeString()`: Determina il tipo di segnale (standard/enum/muxor)
  - `formatValue()`: Formatta i valori dei segnali per la visualizzazione (future)
  - `startReceavingMessages()`: Starts monitoring and receiving the Frames from the can network
  - `updateTable(sgn *acmelib.SignalDecoding, sgnID uint32)`: Given a signal, updates the table for visualizing the received messages
- **model.go**: Inizializza il modello dell'applicazione e gestisce lo stato
  - `newModel()`: Crea un nuovo modello con stato iniziale
  - `NewModelWithDBC()`: Crea un modello con file DBC caricato
  - `tickCmd()`: Gestisce gli eventi di tick per l'aggiornamento del modello
  - `Init()`: Inizializza il modello con i dati del file DBC
- **update.go**: Gestisce gli aggiornamenti del modello in risposta agli eventi
  - `Update()`: Logica per aggiornare lo stato del modello in base agli input dell'utente
- **view.go**: Contiene la logica di rendering delle viste
  - `View()`: Renderizza l'interfaccia utente in base allo stato del modello
  - `filePickerView()`: Visualizza il file picker per la selezione dei file DBC
  - `messageSelectionView()`: Visualizza la lista dei messaggi CAN selezionabili
  - `monitoringView()`: Visualizza la tabella di monitoraggio dei segnali

## Esempio di file DBC supportato

Il tool utilizza acmelib per il parsing dei file DBC. Qualsiasi file DBC standard è supportato.

Esempio con il file `internal/test/MCB.dbc` incluso nel progetto:

```bash
cd /Users/utente/Desktop/can-debug
./can-debug/can-debug
# Naviga a: internal/test/MCB.dbc
```

## Note tecniche

### Visualizzazione segnali DBC

Il tool attualmente mostra la struttura completa dei messaggi selezionati dal file DBC:

- **Nome segnale**: Con unità di misura se disponibili
- **ID messaggio**: ID CAN in formato esadecimale
- **Valore segnale**: Valore attuale del segnale in formato decimale
- **Posizione bit**: Formato "startBit:endBit" per debugging
- **Tipo segnale**: Standard, enum, o muxor con dimensione in bit
- **Navigazione**: Tabella scrollabile con frecce ↑/↓

### Integrazione con dati CAN reali  

Per collegare il tool a dati CAN reali, sostituire `showDBCSignals()` con lettura da bus:

1. **Linux con SocketCAN**:

   ```go
   // In handlers.go, sostituire showDBCSignals() con:
   func (m *Model) readCANData() {
       // Leggere da SocketCAN interface
       // conn, _ := socketcan.Dial("can", "can0")
       // frame, _ := conn.ReadFrame()
       // Usare acmelib decoder per processare i frame
       // decoder := can.NewDecoder(m.Messages)
       // for _, signal := range decoder.Decode(frame) {
       //     // Aggiornare tabella con valori reali
       // }
   }
   ```

## Test

### Test manuale con file DBC di esempio

```bash
# Test con caricamento diretto
./can-debug ./internal/test/MCB.dbc

# Test con file picker (naviga manualmente al file)
./can-debug
```

### Contribuire

1. Fork del progetto
2. Crea un branch per la feature (`git checkout -b feature/nuova-funzione`)
3. Commit delle modifiche (`git commit -am 'Aggiunge nuova funzione'`)
4. Push del branch (`git push origin feature/nuova-funzione`)
5. Apri una Pull Request

## Licenza

Vedi il file LICENSE nel progetto principale.
